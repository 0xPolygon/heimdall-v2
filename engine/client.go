package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/pkg/errors"
)

const (
	forkChoiceUpdatedV2    = "engine_forkchoiceUpdatedV2"
	getPayloadV2           = "engine_getPayloadV2"
	newPayloadV2           = "engine_newPayloadV2"
	exchangeCapabilitiesV2 = "engine_exchangeCapabilitiesV2"
)

type EngineClient struct {
	client *http.Client
	url    string
	reqID  uint64
}

func NewEngineClient(url string, jwtFile string) (*EngineClient, error) {
	secret, err := parseJWTSecretFromFile(jwtFile)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: DefaultRPCTimeout,
		Transport: &MetricsTransport{
			Transport: &JWTTransport{
				Transport: http.DefaultTransport,
				JWTSecret: secret,
			},
		},
	}

	startMetricsServer()

	return &EngineClient{
		client: client,
		url:    url,
	}, nil
}

func (ec *EngineClient) Close() {
	ec.client.CloseIdleConnections()
	if metricsServer != nil {
		metricsServer.Shutdown(context.Background())
	}
}

func (ec *EngineClient) ForkchoiceUpdatedV2(ctx context.Context, state *ForkChoiceState, attrs *PayloadAttributes) (resp *ForkchoiceUpdatedResponse, err error) {
	start := time.Now()
	defer observe(forkChoiceUpdatedV2, start, err)

	var msg json.RawMessage
	msg, err = ec.call(ctx, forkChoiceUpdatedV2, state, attrs)
	if err != nil {
		return
	}
	var data []byte
	data, err = msg.MarshalJSON()
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &resp)
	return
}

func (ec *EngineClient) GetPayloadV2(ctx context.Context, payloadId string) (payload *Payload, err error) {
	start := time.Now()
	defer observe(getPayloadV2, start, err)

	var msg json.RawMessage
	msg, err = ec.call(ctx, getPayloadV2, payloadId)
	if err != nil {
		return
	}
	err = json.Unmarshal(msg, &payload)
	return
}

func (ec *EngineClient) NewPayloadV2(ctx context.Context, payload ExecutionPayload) (resp NewPayloadResponse, err error) {
	start := time.Now()
	defer observe(newPayloadV2, start, err)

	var msg json.RawMessage
	msg, err = ec.call(ctx, "engine_newPayloadV2", payload)
	if err != nil {
		return
	}
	err = json.Unmarshal(msg, &resp)
	return
}

func (ec *EngineClient) CheckCapabilities(ctx context.Context, requiredMethods []string) (err error) {
	start := time.Now()
	defer observe(exchangeCapabilitiesV2, start, err)

	var data []byte
	data, err = ec.call(ctx, "engine_exchangeCapabilities", requiredMethods)
	if err != nil {
		return err
	}
	var response []string
	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	for _, method := range requiredMethods {
		if !contains(response, method) {
			return errors.New(fmt.Sprintf("engine API does not support method '%v'", method))
		}
	}
	return
}

func observe(rpc string, start time.Time, err error) {
	elapsed := time.Since(start)
	rpcCalls.WithLabelValues(rpc).Inc()
	rpcCallDuration.WithLabelValues(rpc).Observe(float64(elapsed.Milliseconds()))
	if err != nil {
		rpcErrors.WithLabelValues(rpc, reflect.TypeOf(err).String()).Inc()
	}
}

func contains(arr []string, val string) bool {
	for _, s := range arr {
		if s == val {
			return true
		}
	}
	return false
}

// Call returns raw response of method call
func (ec *EngineClient) call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	var args []interface{}
	for _, p := range params {
		if p != nil {
			args = append(args, p)
		}
	}

	request := JsonrpcRequest{
		ID:      ec.reqID,
		JSONRPC: "2.0",
		Method:  method,
		Params:  args,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	ec.reqID++

	req, err := http.NewRequestWithContext(ctx, "POST", ec.url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := ec.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	resp := new(JsonrpcResponse)
	if err := json.NewDecoder(response.Body).Decode(resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, *resp.Error
	}

	return resp.Result, nil
}
