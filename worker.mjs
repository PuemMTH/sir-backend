import "./wasm_exec.js";
import wasm from "./app.wasm";

const go = new Go();
const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
go.run(instance);

export default {
  async fetch(req, env, ctx) {
    return handleRequest(req, env, ctx);
  },
};
