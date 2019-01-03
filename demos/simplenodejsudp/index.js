// https://gist.github.com/ryanjon2040/f29787b866316357016971b9c9c363bb

// port to listen to
const PORT = 22222; // Change to your port number
const os = require('os');
const HOST = '0.0.0.0';

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
  console.log('Message received from ' + remote.address + ':' + remote.port + ' - ' + message.toString());

  if (message.toString().toUpperCase().startsWith("PLAYERS")) {
    const number = parseInt(message.toString().split("|")[1].replace("\n",""));
    const postData = { "serverName": process.env.SERVER_NAME, "namespace": process.env.SERVER_NAMESPACE, "playerCount": number };
    sendData(activePlayersMethodURL, postData, function (err, response, body) {
      let serverResponse = message.toString().replace("\n","");
      if (err) {
        console.log(err);
        serverResponse = `${serverResponse}, error in setting Active Players: ${err}\n`;
      } else if (response) {
        console.log("Set Active Players to running OK");
        serverResponse = `${serverResponse}, set Active Players to ${number} OK\n`;
      }
      sendResponse(serverResponse, remote);
    });
  }
  else if (message.toString().toUpperCase().startsWith("HEALTH")) {
    const health = message.toString().split("|")[1].replace("\n","");
    const postData = { "serverName": process.env.SERVER_NAME, "namespace": process.env.SERVER_NAMESPACE, "health": health };
    sendData(healthMethodURL, postData, function (err, response, body) {
      let serverResponse = message.toString().replace("\n","");
      if (err) {
        console.log(err);
        serverResponse = `${serverResponse}, error in setting Server Health: ${err}\n`;
      } else if (response) {
        console.log("Set Server Health OK");
        serverResponse = `${serverResponse}, set Server Status to ${status} OK\n`;
      }
      sendResponse(serverResponse, remote);
    });
  }
  else if (message.toString().toUpperCase().startsWith("STATE")) {
    const state = message.toString().split("|")[1].replace("\n","");
    const postData = { "serverName": process.env.SERVER_NAME, "namespace": process.env.SERVER_NAMESPACE, "state": state };
    sendData(stateMethodURL, postData, function (err, response, body) {
      let serverResponse = message.toString().replace("\n","");
      if (err) {
        console.log(err);
        serverResponse = `${serverResponse}, error in setting Server Status: ${err}\n`;
      } else if (response) {
        console.log("Set Server Status OK");
        serverResponse = `${serverResponse}, set Server Status to ${status} OK\n`;
      }
      sendResponse(serverResponse, remote);
    });
  }
  else if (message.toString().toUpperCase().startsWith("MARKEDFORDELETION")) {
    const markedForDeletion = message.toString().split("|")[1].replace("\n","");
    const postData = { "serverName": process.env.SERVER_NAME, "namespace": process.env.SERVER_NAMESPACE, "markedForDeletion": markedForDeletion };
    sendData(markedForDeletionMethodURL, postData, function (err, response, body) {
      let serverResponse = message.toString().replace("\n","");
      if (err) {
        console.log(err);
        serverResponse = `${serverResponse}, error in setting Server Status: ${err}\n`;
      } else if (response) {
        console.log("Set Server Status OK");
        serverResponse = `${serverResponse}, set Server Status to ${status} OK\n`;
      }
      sendResponse(serverResponse, remote);
    });
  }
  else {
    sendResponse(message.toString(), remote);
  }
});

// Error event. Something bad happened. Prints out error stack and closes the server.
server.on('error', (err) => {
  console.log(`server error:\n${err.stack}`);
  server.close();
});

if (!process.env.SERVER_NAME) {
  console.log("$SERVER_NAME is not defined");
  process.exit(-1);
}

if (!process.env.SERVER_NAMESPACE) {
  console.log("$SERVER_NAMESPACE is not defined");
  process.exit(-1);
}

if (!process.env.API_SERVER_URL) {
  console.log("$API_SERVER_URL is not defined");
  process.exit(-1);
}

if (!process.env.API_SERVER_CODE) {
  console.log("$API_SERVER_CODE is not defined");
  process.exit(-1);
}

const healthMethodURL = `${process.env.API_SERVER_URL}/setsdgshealth?code=${process.env.API_SERVER_CODE}`;
const stateMethodURL = `${process.env.API_SERVER_URL}/setsdgsstate?code=${process.env.API_SERVER_CODE}`;
const activePlayersMethodURL = `${process.env.API_SERVER_URL}/setactiveplayers?code=${process.env.API_SERVER_CODE}`;
const markedForDeletionMethodURL= `${process.env.API_SERVER_URL}/setdgsmarkedfordeletion?code=${process.env.API_SERVER_CODE}`;

const healthPostData = {
  serverName: process.env.SERVER_NAME,
  namespace: process.env.SERVER_NAMESPACE,
  health: "Healthy"
};


// send "Healthy" to the APIServer
sendData(healthMethodURL, healthPostData, function (err, response, body) {
  if (err) {
    console.log(err);
  } else if (response) {
    console.log("Set status Healthy OK");
  }
});

const statePostData = {
  serverName: process.env.SERVER_NAME,
  namespace: process.env.SERVER_NAMESPACE,
  state: "Assigned"
};

// send "Assigned" to the APIServer
sendData(stateMethodURL, statePostData, function (err, response, body) {
  if (err) {
    console.log(err);
  } else if (response) {
    console.log("Set status Assigned OK");
  }
});

// Finally bind our server to the given port and host so that listening event starts happening.
server.bind(PORT, HOST);

function sendData(url, postData, callback) {
  request({
    url: url,
    json: postData,
    method: 'POST',
    maxAttempts: 5, // (default) try 5 times
    retryDelay: 5000, // (default) wait for 5s before trying again
    retryStrategy: request.RetryStrategies.HTTPOrNetworkError // (default) retry on 5xx or network errors
  }, callback);
}

function sendResponse(response, remote) {
  const returnMessage = Buffer.from(`${os.hostname()} says: ${response}`);

  server.send(returnMessage, 0, returnMessage.length, remote.port, remote.address, function (err, bytes) {
    if (err) throw err;
    console.log('UDP message sent to ' + remote.address + ':' + remote.port + '\n');
  });
}