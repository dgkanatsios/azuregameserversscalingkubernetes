// https://gist.github.com/ryanjon2040/f29787b866316357016971b9c9c363bb

// port to listen to
const PORT = 22222; // Change to your port number

const HOST = '127.0.0.1';

// Load datagram module
const dgram = require('dgram');

const request = require('requestretry');

// Create a new instance of dgram socket
const server = dgram.createSocket('udp4');

/**
Once the server is created and binded, some events are automatically created.
We just bind our custom functions to those events so we can do whatever we want.
*/

// Listening event. This event will tell the server to listen on the given address.
server.on('listening', function () {
    var address = server.address();
    console.log('UDP Server listening on ' + address.address + ":" + address.port);
});

// Message event. This event is automatically executed when this server receives a new message
// That means, when we use FUDPPing::UDPEcho in Unreal Engine 4 this event will trigger.
server.on('message', function (message, remote) {
    console.log('Message received from ' + remote.address + ':' + remote.port +' - ' + message.toString());
    server.send(message, 0, message.length, remote.port, remote.address, function(err, bytes) {
	  if (err) throw err;
	  console.log('UDP message sent to ' + remote.address +':'+ remote.port + '\n');
	});
});

// Error event. Something bad happened. Prints out error stack and closes the server.
server.on('error', (err) => {
  console.log(`server error:\n${err.stack}`);
  server.close();
});

const postData = {
  serverName: process.env.SERVER_NAME, 
  status: "Running",
  podNamespace: process.env.POD_NAMESPACE
};

// send "Running" to the APIServer
request({
  url: process.env.SET_SERVER_STATUS_URL,
  json: postData,
  method: 'POST',
  maxAttempts: 5, // (default) try 5 times
  retryDelay: 5000, // (default) wait for 5s before trying again
  retryStrategy: request.RetryStrategies.HTTPOrNetworkError // (default) retry on 5xx or network errors
}, function (err, response, body) {
  // this callback will only be called when the request succeeded or after maxAttempts or on error
  if (err) {
      console.log(err);
  } else if (response) {
      console.log("Set status running OK");
  }
});

// Finally bind our server to the given port and host so that listening event starts happening.
server.bind(PORT, HOST);