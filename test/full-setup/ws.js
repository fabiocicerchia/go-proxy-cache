var https = require('https')
var fs = require('fs')
var WebSocket = require('ws')

function onConnection (ws) {
  ws.on('message', function (message) {
    console.log('Received from client: %s', message)
    ws.send('Server received from client: ' + message)
  })
}

console.log('Server started')

// WS
const ws = new WebSocket.Server({ port: 9001 })
ws.on('connection', onConnection)

// WSS
// Ref: https://github.com/websockets/ws/blob/master/examples/ssl.js
const server = https.createServer({
  cert: fs.readFileSync('./certs/server.pem'),
  key: fs.readFileSync('./certs/server.key'),
  ca: fs.readFileSync('./certs/ca-crt.pem'),
  requestCert: false,
  rejectUnauthorized: false
})

const wss = new WebSocket.Server({ server })
wss.on('connection', onConnection)
server.listen(9002)
