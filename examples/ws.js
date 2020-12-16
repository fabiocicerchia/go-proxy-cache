var https = require('https');
var fs = require('fs');
var WebSocket = require('ws');

console.log("Server started");

// WS
ws = new WebSocket.Server({port: 9001});
ws.on('connection', function(ws) {
    ws.on('message', function(message) {
      console.log('Received from client: %s', message);
      ws.send('Server received from client: ' + message);
  });
});

// WSS
// Ref: https://github.com/websockets/ws/blob/master/examples/ssl.js
const server = https.createServer({
  cert: fs.readFileSync('./server.pem'),
  key: fs.readFileSync('./server.key'),
  ca: fs.readFileSync('./ca-crt.pem'),
  requestCert: false,
  rejectUnauthorized: false
});
wss = new WebSocket.Server({ server });
wss.on('connection', function(wss) {
  wss.on('message', function(message) {
    console.log('Received from client: %s', message);
    wss.send('Server received from client: ' + message);
  });
});
server.listen(9002);
