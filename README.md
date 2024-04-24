Websocket Channels Demo
=======================

This is a sample application demonstrating subscribable websockets

Client 1 can subscribe to Client 2.  Client 3 can subscribe to Client 1 and 2.  All the way to Client N.

If you're not familiar with compiling Go programs, I've included a [video demo](https://www.loom.com/share/95e063e6b7f84998bbeab4364ff2a9e7) for your convenience.

Usage
-----

Run the command:

    go run .

Then open up different browsers (i.e. Chrome, Chrome incognito, Firefox, Edge) to simular different client.

Different tabs in the same browser will not work due to cookie based sessions.