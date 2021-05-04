Minimal gRPC Server
===================
This library wraps Google's gRPC server in Golang to make running the server synchronous, with a stop instruction sent to the server by receiving a signal (e.g. SIGTERM, SIGQUIT).
