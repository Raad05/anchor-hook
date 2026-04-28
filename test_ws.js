const WebSocket = require('ws');
const ws = new WebSocket('ws://127.0.0.1:8900');
ws.on('open', () => {
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'logsSubscribe',
    params: [{"mentions": ["DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD"]}, {"commitment": "confirmed"}]
  }));
});
ws.on('message', data => {
  console.log('WS msg:', data.toString());
});
