Websocket Channels Demo
=======================

This is a sample application demonstrating subscribable web sockets

It supports Client N to many connections. Client 1 can subscribe to Client 2.  Client 3 can subscribe to Clients 1, 2, and N.

If compiling Go programs is unfamiliar, I've included a [video demo](https://www.loom.com/share/95e063e6b7f84998bbeab4364ff2a9e7) for your convenience.

Usage
-----

Run the command:

    go run .

Then open up different browsers (i.e. Chrome, Chrome incognito, Firefox, Edge) to simulate a different client.

Different tabs in the same browser will not work due to cookie-based sessions.
