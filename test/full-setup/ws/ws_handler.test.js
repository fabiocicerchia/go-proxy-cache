"use strict";
const { test, beforeEach, afterEach, mock } = require("node:test");
const assert = require("node:assert");
const { EventEmitter } = require("node:events");

const { createOnConnection } = require("./ws_handler");

function setup() {
  const calls = { extract: [], startSpan: [], events: [], ended: 0 };
  const api = {
    context: { active: () => "ACTIVE_CTX" },
    defaultTextMapGetter: "GETTER",
  };
  const propagator = {
    extract: (ctx, carrier, getter) => {
      calls.extract.push({ ctx, carrier, getter });
      return "EXTRACTED_CTX";
    },
  };
  const tracer = {
    startSpan: (name, opts, ctx) => {
      calls.startSpan.push({ name, opts, ctx });
      return {
        addEvent: (e) => calls.events.push(e),
        end: () => {
          calls.ended++;
        },
      };
    },
  };
  const ws = new EventEmitter();
  ws.sent = [];
  ws.send = (data) => ws.sent.push(data);
  const request = { headers: { traceparent: "00-abc-def-01" } };

  createOnConnection({ tracer, propagator, api })(ws, request);
  return { ws, request, calls };
}

beforeEach(() => mock.method(console, "log", () => {}));
afterEach(() => mock.restoreAll());

test("replies to each message with a prefixed echo", () => {
  const { ws } = setup();
  ws.emit("message", "{}");
  ws.emit("message", "hello");
  assert.deepStrictEqual(ws.sent, [
    "Server received from client: {}",
    "Server received from client: hello",
  ]);
});

test("extracts trace context from request headers", () => {
  const { ws, request, calls } = setup();
  ws.emit("message", "x");
  assert.deepStrictEqual(calls.extract, [
    { ctx: "ACTIVE_CTX", carrier: request.headers, getter: "GETTER" },
  ]);
});

test("starts span in extracted context, records event and ends it", () => {
  const { ws, calls } = setup();
  ws.emit("message", "x");
  assert.deepStrictEqual(calls.startSpan, [
    { name: "http_request", opts: {}, ctx: "EXTRACTED_CTX" },
  ]);
  assert.deepStrictEqual(calls.events, ["data_received"]);
  assert.strictEqual(calls.ended, 1);
});

test("does nothing until a message arrives", () => {
  const { ws, calls } = setup();
  assert.deepStrictEqual(ws.sent, []);
  assert.strictEqual(calls.startSpan.length, 0);
});

test("one span per message", () => {
  const { ws, calls } = setup();
  ws.emit("message", "a");
  ws.emit("message", "b");
  assert.strictEqual(calls.startSpan.length, 2);
  assert.strictEqual(calls.ended, 2);
});
