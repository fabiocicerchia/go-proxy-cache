"use strict";

function createOnConnection({ tracer, propagator, api }) {
  return function onConnection(ws, request) {
    ws.on("message", function (message) {
      console.log(request.headers);
      const currentContext = propagator.extract(
        api.context.active(),
        request.headers,
        api.defaultTextMapGetter,
      );
      const span = tracer.startSpan("http_request", {}, currentContext);

      span.addEvent("data_received");
      console.log("Received from client: %s", message);
      ws.send("Server received from client: " + message);

      span.end();
    });
  };
}

module.exports = { createOnConnection };
