import * as grpc from '@grpc/grpc-js';
import * as log from 'loglevel';
import { Result, ok, err } from 'neverthrow';
import { setTimeout } from "timers/promises";

const BIND_IP: string = "0.0.0.0";
const MILLIS_IN_SECOND: number = 1000;
const INTERRUPT_SIGNAL: string = "SIGNINT";
const QUIT_SIGNAL: string = "SIGQUIT";
const TERM_SIGNAL: string = "SIGTERM";

// ====================================== NOTE ========================================================= 
// All this rigamarole is necessary due to gRPC's stupid "unimplemented server" requirements
// See:
// - https://github.com/agreatfool/grpc_tools_node_protoc_ts/issues/79
// - https://github.com/agreatfool/grpc_tools_node_protoc_ts/blob/master/doc/server_impl_signature.md
//
// ====================================== NOTE =========================================================
type KnownKeys<T> = {
    [K in keyof T]: string extends K ? never : number extends K ? never : K
} extends { [_ in keyof T]: infer U } ? U : never;

export type KnownKeysOnly<T extends Record<any, any>> = Pick<T, KnownKeys<T>>;

export class TypedServerOverride extends grpc.Server {
    addTypedService<TypedServiceImplementation extends Record<any,any>>(service: grpc.ServiceDefinition, implementation: TypedServiceImplementation): void {
        this.addService(service, implementation);
    }
}

export class MinimalGRPCServer {
    private readonly listenPort: number;
    private readonly stopGracePeriodSeconds: number;    // How long we'll give the server to stop after asking nicely before we kill it
    private readonly serviceRegistrationFuncs: { (server: TypedServerOverride): void; }[]

    // Creates a minimal gRPC server but doesn't start it
    // The service registration funcs will be applied, in order, to register services with the underlying gRPC server object
    constructor(
        listenPort: number,
        stopGracePeriodSeconds: number,
        // NOTE: Users wanting to register services declared with class style should use server.addTypedService,
        //  and pass in a class that implements KnownKeysOnly<IYourServiceServer>
        serviceRegistrationFuncs: { (server: TypedServerOverride): void; }[]
    ) {
        this.listenPort = listenPort;
        this.stopGracePeriodSeconds = stopGracePeriodSeconds;
        this.serviceRegistrationFuncs = serviceRegistrationFuncs;
    }

    // Runs the server synchronously until an interrupt signal is received
    public async runUntilInterrupted(): Promise<Result<null, Error>> {
        // Signals are used to interrupt the server, so we catch them here
        const signalsToHandle: Array<string> = [INTERRUPT_SIGNAL, QUIT_SIGNAL, TERM_SIGNAL];
        const signalReceivedPromises: Array<Promise<null>> = signalsToHandle.map((signal) => {
            return new Promise((resolve, _unusedReject) => {
                process.on(signal, () => {
                    resolve(null);
                });
            });
        });
        const anySignalReceivedPromise: Promise<null> = Promise.race(signalReceivedPromises);
        const runResult: Result<null, Error> = await this.runUntilStopped(anySignalReceivedPromise);
        if (runResult.isErr()) {
            return err(runResult.error);
        }
        return ok(null);
    }

    // Runs the server synchronously until the given promise is resolved
    public async runUntilStopped(stopper: Promise<null>): Promise<Result<null, Error>> {
        // NOTE: This is where we'd want to add server call interceptors to log the request & response...
        // ...but they're not supported: https://github.com/grpc/grpc-node/issues/419
        // As of 2021-09-20, this is a difference from the Go version!
        const grpcServer: TypedServerOverride = new TypedServerOverride();

        for (let registrationFunc of this.serviceRegistrationFuncs) {
            registrationFunc(grpcServer);
        }

        const listenUrl: string = BIND_IP + ":" + this.listenPort;
        const bindPortPromise: Promise<Result<number, Error>> = new Promise((resolve) => {
            grpcServer.bindAsync(listenUrl, grpc.ServerCredentials.createInsecure(), (error: Error | null, portNumber: number) => {
                if (error === null) {
                    resolve(ok(portNumber));
                } else {
                    resolve(err(error));
                }
            });
        })
        const bindPortResult = await bindPortPromise;
        if (bindPortResult.isErr()) {
            return err(bindPortResult.error);
        }
        grpcServer.start();

        await stopper;

        const tryShutdownPromise: Promise<Result<null, Error>> = new Promise((resolve, _unusedReject) => {
            grpcServer.tryShutdown(() => {
                resolve(ok(null));
            })
        })
        const timeoutAbortController: AbortController = new AbortController();
        const timeoutPromise: Promise<Result<null, Error>> = setTimeout(
            this.stopGracePeriodSeconds * MILLIS_IN_SECOND,
            err(new Error("gRPC server failed to stop gracefully after waiting for " + this.stopGracePeriodSeconds + "s")),
            {
                signal: timeoutAbortController.signal,
            },
        );
        const gracefulShutdownResult: Result<null, Error> = await Promise.race([tryShutdownPromise, timeoutPromise]);
        if (gracefulShutdownResult.isOk()) {
            log.debug("gRPC server has exited gracefully");
        } else {
            log.warn("gRPC server failed to stop gracefully after " + this.stopGracePeriodSeconds + "s; hard-stopping now...");
            grpcServer.forceShutdown();
            log.debug("gRPC server was forcefully stopped");
        }
        // If the timeout is still running (i.e. in the event of a successful shutdown), kill the timeout thread
        //  so that the Node engine doesn't block
        timeoutAbortController.abort();
        return ok(null);
    }
}
