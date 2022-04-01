const https = require('https')
const fs = require('fs')
const WebSocket = require('ws')

const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');
const { AlwaysOnSampler, W3CTraceContextPropagator } = require("@opentelemetry/core")
const { WebTracerProvider } = require('@opentelemetry/web');
const { SimpleSpanProcessor } = require('@opentelemetry/tracing');
const { JaegerExporter } = require('@opentelemetry/exporter-jaeger');
const api = require('@opentelemetry/api');

const tracerProvider = new WebTracerProvider({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: 'ws-server',
  }),
  sampler: new AlwaysOnSampler()
});
tracerProvider.addSpanProcessor(new SimpleSpanProcessor(new JaegerExporter({
  endpoint: 'http://jaeger:14268/api/traces'
})));
const propagator = new W3CTraceContextPropagator()

// Register the tracer
tracerProvider.register();
const tracer = tracerProvider.getTracer('ws-server');

function onConnection (ws, request) {
  ws.on('message', function (message) {
    console.log(request.headers);
    const currentContext = propagator.extract(api.context.active(), request.headers, api.defaultTextMapGetter);
    const span = tracer.startSpan('http_request', {}, currentContext);

    span.addEvent('data_received')
    console.log('Received from client: %s', message)
    ws.send('Server received from client: ' + message)

    span.end()
  })
}

console.log('Server started')

// WS
const ws = new WebSocket.Server({ port: 9001 })
ws.on('connection', onConnection)

// WSS
// Ref: https://github.com/websockets/ws/blob/master/examples/ssl.js
const server = https.createServer({
  cert: fs.readFileSync('./certs/default/server.pem'),
  key: fs.readFileSync('./certs/default/server.key'),
  ca: fs.readFileSync('./certs/default/ca-crt.pem'),
  requestCert: false,
  rejectUnauthorized: false
})

const wss = new WebSocket.Server({ server })
wss.on('connection', onConnection)
server.listen(9002)
