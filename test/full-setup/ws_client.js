'use strict'
const WebSocket = require('ws')
var initTracer = require('jaeger-client').initTracer

var tracer = initTracer({
  serviceName: 'ws-client',
  sampler: {
    type: "const",
    param: 1
  },
  reporter: {
    collectorEndpoint: 'http://jaeger:14268/api/traces',
  }
}, {})

// WS --------------------------------------------------------------------------

// const socket = new WebSocket('ws://testing.local:9001') // direct
// const socket = new WebSocket('ws://testing.local:40081') // nginx
const socket = new WebSocket('ws://testing.local:50080') // go-proxy-cache

const spanPlain = tracer.startSpan('ws_plain');

console.log('launched plain')
socket.onopen = function (event) {
  spanPlain.log({'event': 'data_sent'});
  console.log('Sending plain message')
  socket.send('{}')
}

socket.onmessage = function (event) {
  spanPlain.log({'event': 'data_reiceved'});
  console.log(event.data)
}
socket.on('error', function (err) {
  console.log(err)
  spanPlain.setTag(opentracing.Tags.ERROR, true)
  spanPlain.log({'event': 'error', 'error.object': err, 'message': err.message, 'stack': err.stack})
  spanPlain.finish()
})

// WSS -------------------------------------------------------------------------

const opts = {
  rejectUnauthorized: false
}

// const socket2 = new WebSocket('wss://testing.local:9002', opts) // direct
// const socket2 = new WebSocket('wss://testing.local:40082', opts) // nginx
const socket2 = new WebSocket('wss://testing.local:50443', opts) // go-proxy-cache
const spanSecure = tracer.startSpan('ws_secure');

console.log('launched secure')
socket2.onopen = function (event) {
  spanSecure.log({'event': 'data_sent'});
  console.log('Sending secure message')
  socket2.send('{}')
}

socket2.onmessage = function (event) {
  spanSecure.log({'event': 'data_reiceved'});
  console.log(event.data)

  console.log('ALL GOOD, EXITING...')
  process.exit(0)
}
socket2.on('error', function (event) {
  console.log(event)
  spanSecure.setTag(opentracing.Tags.ERROR, true)
  spanSecure.log({'event': 'error', 'error.object': err, 'message': err.message, 'stack': err.stack})
  spanSecure.finish()

  console.log('ERROR, EXITING...')
  process.exit(1)
})
