package com.nmidsdk.worker;

/**
 * A registered worker function. Mirrors sdk/node/lib/worker/function.js.
 * <p>
 * The function receives the decoded job {@link Response} and must return the
 * ret payload (typically a msgpack-encoded RetStruct via
 * {@link com.nmidsdk.model.utils.Utils#buildRetStruct}). It may throw to
 * signal failure.
 * <p>
 * Implementations run on the worker's job thread pool, so blocking work
 * (e.g. calling an external HTTP API) is fine and will not block the reader
 * thread that receives jobs from the server.
 */
@FunctionalInterface
public interface Function {
    byte[] run(Response resp) throws Exception;
}
