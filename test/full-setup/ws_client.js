'use strict'
const WebSocket = require('ws')

// WS --------------------------------------------------------------------------

// const socket = new WebSocket('ws://testing.local:9001') // direct
// const socket = new WebSocket('ws://testing.local:40081') // nginx
const socket = new WebSocket('ws://testing.local:50080') // go-proxy-cache

console.log('launched plain')
socket.onopen = function (event) {
  console.log('Sending plain message')
  socket.send('{}')
}

socket.onmessage = function (event) {
  console.log(event.data)
}
socket.on('error', function (event) {
  console.log(event)
})

// WSS -------------------------------------------------------------------------

const opts = {
  rejectUnauthorized: false
}

// const socket2 = new WebSocket('wss://testing.local:9002', opts) // direct
// const socket2 = new WebSocket('wss://testing.local:40082', opts) // nginx
const socket2 = new WebSocket('wss://testing.local:50443', opts) // go-proxy-cache

console.log('launched secure')
socket2.onopen = function (event) {
  console.log('Sending secure message')
  socket2.send('{}')
}

socket2.onmessage = function (event) {
  console.log(event.data)
  console.log('ALL GOOD, EXITING...')
  process.exit(0)
}
socket2.on('error', function (event) {
  console.log(event)
  console.log('ERROR, EXITING...')
  process.exit(1)
})
