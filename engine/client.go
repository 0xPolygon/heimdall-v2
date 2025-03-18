package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	gethEngine "github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/pkg/errors"
)

type EngineClient struct {
	client *http.Client
	url    string
	reqID  uint64
}

type JsonrpcRequest struct {
	ID      uint64        `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JsonrpcResponse struct {
	ID      int             `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *EthError       `json:"error"`
}

type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type EthError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err EthError) Error() string {
	return fmt.Sprintf("Error %d (%s)", err.Code, err.Message)
}

var _ ExecutionEngineClient = (*EngineClient)(nil)

func NewEngineClient(url string, jwtFile string) (*EngineClient, error) {
	secret, err := parseJWTSecretFromFile(jwtFile)
	if err != nil {
		return nil, err
	}
	authTransport := &jwtTransport{
		underlyingTransport: http.DefaultTransport,
		jwtSecret:           secret,
	}
	client := &http.Client{
		Timeout:   DefaultRPCTimeout,
		Transport: authTransport,
	}
	return &EngineClient{
		client: client,
		url:    url,
	}, nil
}

func (ec *EngineClient) Close() {
	ec.client.CloseIdleConnections()
}

func (ec *EngineClient) ForkchoiceUpdatedV2(ctx context.Context, state *gethEngine.ForkchoiceStateV1, attrs *gethEngine.PayloadAttributes) (*gethEngine.ForkChoiceResponse, error) {
	msg, err := ec.call(ctx, "engine_forkchoiceUpdatedV2", state, attrs)
	if err != nil {
		return nil, err
	}
	data, err := msg.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var response gethEngine.ForkChoiceResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (ec *EngineClient) GetPayloadV2(ctx context.Context, payloadId string) (*gethEngine.ExecutionPayloadEnvelope, error) {
	msg, err := ec.call(ctx, "engine_getPayloadV2", payloadId)
	if err != nil {
		return nil, err
	}
	var response gethEngine.ExecutionPayloadEnvelope
	err = json.Unmarshal(msg, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (ec *EngineClient) NewPayloadV2(ctx context.Context, payload gethEngine.ExecutableData) (*gethEngine.PayloadStatusV1, error) {
	msg, err := ec.call(ctx, "engine_newPayloadV2", payload)
	if err != nil {
		return nil, err
	}
	var response gethEngine.PayloadStatusV1
	err = json.Unmarshal(msg, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (ec *EngineClient) CheckCapabilities(ctx context.Context, requiredMethods []string) error {
	data, err := ec.call(ctx, "engine_exchangeCapabilities", requiredMethods)
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
	return nil
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
