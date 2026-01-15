//go:build js && wasm

package rulesengine

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
)

type WasmInput struct {
	Company    Company          `json:"company"`
	User       User             `json:"user"`
	Flag       Flag             `json:"flag"`
	Usage      *int64           `json:"usage,omitempty"`
	EventUsage map[string]int64 `json:"event_usage,omitempty"`
}

type WasmOutput struct {
	Result *CheckFlagResult `json:"result"`
	Error  string           `json:"error,omitempty"`
}

func checkFlag(this js.Value, args []js.Value) interface{} {
	input := args[0].String()

	var wasmInput WasmInput
	json.Unmarshal([]byte(input), &wasmInput)

	var opts []CheckFlagOption
	if wasmInput.Usage != nil {
		opts = append(opts, WithUsage(*wasmInput.Usage))
	}
	for eventSubtype, quantity := range wasmInput.EventUsage {
		opts = append(opts, WithEventUsage(eventSubtype, quantity))
	}

	result, err := CheckFlag(context.Background(), &wasmInput.Company, &wasmInput.User, &wasmInput.Flag, opts...)

	output := WasmOutput{
		Result: result,
		Error:  "",
	}
	if err != nil {
		output.Error = err.Error()
	}

	outputBytes, _ := json.Marshal(output)
	return string(outputBytes)
}

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("checkFlag", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic: %v\n", r)
			}
		}()

		return checkFlag(this, args)
	}))

	// go program stays running
	<-c
}
