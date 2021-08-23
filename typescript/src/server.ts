import * as grpc from 'grpc';
import * as log from 'loglevel';
import { Result, ok, err } from 'neverthrow';

const MILLIS_IN_SECOND: number = 1000;
const INTERRUPT_SIGNAL: string = "SIGNINT";
const QUIT_SIGNAL: string = "SIGQUIT";
const TERM_SIGNAL: string = "SIGTERM";

export class MinimalGRPCServer {
    private readonly listenPort: number;
    private readonly stopGracePeriodSeconds: number;    // How long we'll give the server to stop after asking nicely before we kill it
    private readonly serviceRegistrationFuncs: { (server: grpc.Server): void; }[]

    // Creates a minimal gRPC server but doesn't start it
    // The service registration funcs will be applied, in order, to register services with the underlying gRPC server object
    constructor(
        listenPort: number,
        stopGracePeriodSeconds: number,
        serviceRegistrationFuncs: { (server: grpc.Server): void; }[]
    ) {
        this.listenPort = listenPort;
        this.stopGracePeriodSeconds = stopGracePeriodSeconds;
        this.serviceRegistrationFuncs = serviceRegistrationFuncs;
    }

    async run(): Promise<Result<null, Error>> {
        const grpcServer: grpc.Server = new grpc.Server();

        for (let registrationFunc of this.serviceRegistrationFuncs) {
            registrationFunc(grpcServer);
        }

        const listenUrl: string = ":" + this.listenPort;
        const boundPort: number = grpcServer.bind(listenUrl, grpc.credentials.createInsecure());
        if (boundPort === 0) {
            return err(new Error("An error occurred binding the server to listen URL '"+ boundPort +"'"));
        }

        // Signals are used to interrupt the server, so we catch them here
        const signalsToHandle: Array<string> = [INTERRUPT_SIGNAL, QUIT_SIGNAL, TERM_SIGNAL];
        const signalReceivedPromises: Array<Promise<Result<null, Error>>> = signalsToHandle.map((signal) => {
            return new Promise((resolve, _unusedReject) => {
                process.on(signal, () => {
                    resolve(ok(null));
                });
            });
        });
        const anySignalReceivedPromise: Promise<Result<null, Error>> = Promise.race(signalReceivedPromises);

        grpcServer.start();

        await anySignalReceivedPromise;

        const tryShutdownPromise: Promise<Result<null, Error>> = new Promise((resolve, _unusedReject) => {
            grpcServer.tryShutdown(() => {
                resolve(ok(null));
            })
        })
        const timeoutPromise: Promise<Result<null, Error>> = new Promise((resolve, _unusedReject) => {
            setTimeout(
                () => {
                    resolve(err(new Error("gRPC server failed to stop gracefully after waiting for " + this.stopGracePeriodSeconds + "s")));
                },
                this.stopGracePeriodSeconds * MILLIS_IN_SECOND;
            );
        })
        const gracefulShutdownResult: Result<null, Error> = await Promise.race([tryShutdownPromise, timeoutPromise]);
        if (gracefulShutdownResult.isErr()) {
            log.debug("gRPC server has exited gracefully");
        } else {
            log.warn("gRPC server failed to stop gracefully after " + this.stopGracePeriodSeconds + "s; hard-stopping now...");
            grpcServer.forceShutdown();
            log.debug("gRPC server was forcefully stopped");
        }

        return ok(null);
    }
}
