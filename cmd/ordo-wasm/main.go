//go:build js && wasm

// Command ordo-wasm is internal/core for the web app, built with GOOS=js GOARCH=wasm: the calls
// native/core binds for the Android app, JSON strings in and out. It sets globalThis.ordoCore, then
// calls globalThis.ordoReady when that is a function.
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
		return held(s.Open([]byte(args[0].String())))
	}))
	api.Set("import", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return held(s.Import([]byte(args[0].String())))
	}))
	api.Set("perform", js.FuncOf(func(_ js.Value, args []js.Value) any {
		var hidden []string
		if len(args) > 2 && args[2].Type() == js.TypeString {
			_ = json.Unmarshal([]byte(args[2].String()), &hidden)
		}
		return answer(s.PerformIn(args[0].String(), []byte(args[1].String()), hidden))
	}))
	api.Set("replay", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return answer(s.Replay(args[0].String(), []byte(args[1].String())))
	}))
	api.Set("project", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return text(s.Project(args[0].String()))
	}))
	api.Set("export", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return text(s.Export(args[0].String()))
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

func held(id string, err error) any {
	if err != nil {
		return answer(map[string]string{"error": err.Error()})
	}
	return answer(map[string]string{"id": id})
}

func text(raw []byte) any {
	if raw == nil {
		return nil
	}
	return string(raw)
}

func answer(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
