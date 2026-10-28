//go:build js && wasm

// Command ordo-core is internal/core for the phone and browser app, built with GOOS=js GOARCH=wasm.
// It sets globalThis.ordoCore, then calls globalThis.ordoReady when that is a function.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/ilyasturki/ordo/internal/core"
)

func main() {
	s := core.New()
	api := js.Global().Get("Object").New()
	api.Set("open", js.FuncOf(func(_ js.Value, args []js.Value) any {
		id, err := s.Open([]byte(args[0].String()))
		if err != nil {
			return answer(map[string]string{"error": err.Error()})
		}
		return answer(map[string]string{"id": id})
	}))
	api.Set("perform", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return answer(s.Perform(args[0].String(), []byte(args[1].String())))
	}))
	api.Set("replay", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return answer(s.Replay(args[0].String(), []byte(args[1].String())))
	}))
	api.Set("project", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if raw := s.Project(args[0].String()); raw != nil {
			return string(raw)
		}
		return nil
	}))
	api.Set("close", js.FuncOf(func(_ js.Value, args []js.Value) any {
		s.Close(args[0].String())
		return nil
	}))
	js.Global().Set("ordoCore", api)
	if ready := js.Global().Get("ordoReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}
	select {}
}

func answer(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
