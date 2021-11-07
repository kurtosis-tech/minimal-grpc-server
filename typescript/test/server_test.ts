import assert from "assert";
import { expect } from "chai";
import { Result } from 'neverthrow';
import { MinimalGRPCServer, TypedServerOverride } from "../src/server";
import { setTimeout } from "timers/promises";

const LISTEN_PORT: number = 9003;
const STOP_GRACE_PERIOD_SECONDS: number = 10;

describe(MinimalGRPCServer.name, function() {
    describe('#runUntilStopped', function() {
        it('should start the server, let it run for a bit, then stop it without issue', async function() {
            const serviceRegistrationFuncs: { (server: TypedServerOverride): void; }[] = [];
            const server = new MinimalGRPCServer(LISTEN_PORT, STOP_GRACE_PERIOD_SECONDS, serviceRegistrationFuncs);

            var stopFunction;
            const stopPromise: Promise<null> = new Promise((resolve) => { stopFunction = resolve; })
            const serverResultPromise: Promise<Result<null, Error>> = server.runUntilStopped(stopPromise);

            const firstTimeoutPromise: Promise<null> = setTimeout(1000, null);
            const firstRaceResult: Result<null, Error> | undefined = await Promise.race([serverResultPromise, firstTimeoutPromise]);
            expect(firstRaceResult).to.equal(undefined, `The server unexpectedly returned a result before we sent a signal to stop it:\n${firstRaceResult}`);
            
            stopFunction();

            const maxWaitForServerToStopMillis: number = 1000;
            const secondTimeoutPromise: Promise<null> = setTimeout(maxWaitForServerToStopMillis, null);
            const secondRaceResult: Result<null, Error> | undefined = await Promise.race([serverResultPromise, secondTimeoutPromise]);
            if (secondRaceResult === undefined) {
                assert.fail(`Expected the server to have stopped after ${maxWaitForServerToStopMillis}ms, but it didn't`);
            }
            if (secondRaceResult.isErr()) {
                assert.fail(`Expected the server to exit without an error but it threw the following:\n${secondRaceResult.error}`);
            }
        })
    })
});