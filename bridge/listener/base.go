package listener

import (
	"context"
	"math/big"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/RichardKnop/machinery/v1/tasks"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/0xPolygon/heimdall-v2/bridge/queue"
	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

// Listener is a chain event listener run by the listener service.
type Listener interface {
	Start() error
	Stop()
	String() string
}

// headerProcessor handles the block headers a BaseListener polls for.
type headerProcessor interface {
	ProcessHeader(*blockHeader)
}

type BaseListener struct {
	Logger log.Logger
	name   string

	// The "subclass" of BaseService
	impl headerProcessor

	chainClient *ethclient.Client

	// header channel
	HeaderChannel chan *blockHeader

	// cancel function for poll/subscription
	cancelSubscription context.CancelFunc

	// header listener subscription
	cancelHeaderProcess context.CancelFunc

	// cli context
	cliCtx client.Context

	// task queue the listener enqueues processor tasks on
	taskQueue taskQueue

	// http client to subscribe to
	httpClient *rpchttp.HTTP

	// storage client
	storageClient *leveldb.DB
}

// taskQueue is the part of the machinery server listeners use to enqueue tasks.
type taskQueue interface {
	SendTask(signature *tasks.Signature) (*result.AsyncResult, error)
	IsTaskRegistered(name string) bool
}

const (
	rootChainTaskRetryCount = 5
	blockTaskRetryCount     = 3
)

// unregisteredTaskWarned records the task names sendTask has already warned
// about skipping, so a disabled processor is reported once per process.
var unregisteredTaskWarned sync.Map

type blockHeader struct {
	header      *types.Header // block header
	isFinalized bool          // if the block is a finalized block or not
}

// NewBaseListener creates a new BaseListener.
func NewBaseListener(cliCtx client.Context, queueConnector *queue.Connector, httpClient *rpchttp.HTTP, chainClient *ethclient.Client, name string, impl headerProcessor) *BaseListener {
	return &BaseListener{
		Logger:        util.Logger().With("service", "listener", "module", name),
		name:          name,
		impl:          impl,
		storageClient: util.GetBridgeDBInstance(viper.GetString(util.BridgeDBFlag)),

		cliCtx:      cliCtx,
		taskQueue:   queueConnector.Server,
		httpClient:  httpClient,
		chainClient: chainClient,

		HeaderChannel: make(chan *blockHeader),
	}
}

// String implements Service by returning a string representation of the service.
func (bl *BaseListener) String() string {
	return bl.name
}

// StartHeaderProcess starts the header process when they get a new header
func (bl *BaseListener) StartHeaderProcess(ctx context.Context) {
	bl.Logger.Info("BaseListener: starting header process")

	for {
		select {
		case newHeader := <-bl.HeaderChannel:
			bl.impl.ProcessHeader(newHeader)
		case <-ctx.Done():
			bl.Logger.Info("BaseListener: header process stopped")
			return
		}
	}
}

// StartPolling starts polling
func (bl *BaseListener) StartPolling(ctx context.Context, pollInterval time.Duration, number *big.Int) {
	// How often to fire the passed in function in second
	interval := pollInterval

	// Set up the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)

	// start listening
	for {
		select {
		case <-ticker.C:
			var bHeader *blockHeader

			header, err := bl.chainClient.HeaderByNumber(ctx, number)
			if err == nil && header != nil {
				if number != nil {
					// finalized was requested
					bHeader = &blockHeader{header: header, isFinalized: true}
				} else {
					// the latest was requested
					bHeader = &blockHeader{header: header, isFinalized: false}
				}
			}

			// if an error occurred and finalized was requested, fall back to the latest block
			if err != nil && number != nil {
				header, err = bl.chainClient.HeaderByNumber(ctx, nil)
				if err == nil && header != nil {
					bHeader = &blockHeader{header: header, isFinalized: false}
				}
			}

			if err != nil {
				bl.Logger.Error("BaseListener: error in fetching block header while polling", "err", err)
			}

			// push data to the channel
			if bHeader != nil {
				bl.HeaderChannel <- bHeader
			}
		case <-ctx.Done():
			bl.Logger.Info("BaseListener: polling stopped")
			ticker.Stop()

			return
		}
	}
}

// sendTask enqueues taskName to run no earlier than delay from now. A task no
// processor registered is skipped: the queue would redeliver it forever.
func (bl *BaseListener) sendTask(taskName string, retryCount int, delay time.Duration, args ...tasks.Arg) error {
	if !bl.taskQueue.IsTaskRegistered(taskName) {
		if _, warned := unregisteredTaskWarned.LoadOrStore(taskName, struct{}{}); !warned {
			bl.Logger.Warn("BaseListener: not queueing tasks with no registered processor; is the processor disabled by --only?", "taskName", taskName)
		}
		return nil
	}

	eta := time.Now().Add(delay)
	signature := &tasks.Signature{Name: taskName, Args: args, RetryCount: retryCount, ETA: &eta}

	bl.Logger.Info("BaseListener: sending task", "taskName", taskName, "eta", eta)

	if _, err := bl.taskQueue.SendTask(signature); err != nil {
		bl.Logger.Error("BaseListener: error sending task", "taskName", taskName, "error", err)
		return err
	}

	return nil
}

func stringArg(value string) tasks.Arg {
	return tasks.Arg{Type: "string", Value: value}
}

// Stop stops all necessary go routines
func (bl *BaseListener) Stop() {
	// cancel subscription if any
	if bl.cancelSubscription != nil {
		bl.cancelSubscription()
	}

	// cancel the header process
	if bl.cancelHeaderProcess != nil {
		bl.cancelHeaderProcess()
	}
}
