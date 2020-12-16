'use strict';
var WebSocket = require('ws');

// WS --------------------------------------------------------------------------

// let socket = new WebSocket("ws://testing.local:9001"); // direct
// let socket = new WebSocket("ws://testing.local:8081"); // nginx
let socket = new WebSocket("ws://testing.local:80"); // go-proxy-cache

console.log("launched plain");
socket.onopen = function (event) {
  console.log("Sending plain message");
  socket.send('{}');
};

socket.onmessage = function (event) {
  console.log(event.data);
}
socket.on('error', function(event) {
  console.log(event);
});

// WSS -------------------------------------------------------------------------

var opts = {
  rejectUnauthorized: false
};

// let socket2 = new WebSocket("wss://testing.local:9002", opts); // direct
// let socket2 = new WebSocket("wss://testing.local:8082", opts); // nginx
let socket2 = new WebSocket("wss://testing.local:443", opts); // go-proxy-cache

console.log("launched secure");
socket2.onopen = function (event) {
  console.log("Sending secure message");
  socket2.send('{}');
};

socket2.onmessage = function (event) {
  console.log(event.data);
  process.exit(0);
}
socket2.on('error', function(event) {
  console.log(event);
  process.exit(1);
});
