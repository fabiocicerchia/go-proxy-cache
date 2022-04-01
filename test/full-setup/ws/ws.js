const https = require('https')
const fs = require('fs')
const WebSocket = require('ws')
const opentracing = require('opentracing')
const initTracer = require('jaeger-client').initTracer

const tracer = initTracer({
  serviceName: 'ws-server',
  sampler: {
    type: 'const',
    param: 1
  },
  reporter: {
    collectorEndpoint: 'http://jaeger:14268/api/traces',
  }
}, {})

function onConnection (ws, request) {
  const headersCarrier = request.headers

  ws.on('message', function (message) {
    console.log(request.headers);
    const wireCtx = tracer.extract(opentracing.FORMAT_HTTP_HEADERS, headersCarrier)
    const span = tracer.startSpan('http_request', { childOf: wireCtx })

    span.log({ event: 'data_received' })
    console.log('Received from client: %s', message)
    ws.send('Server received from client: ' + message)

    span.finish()
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
