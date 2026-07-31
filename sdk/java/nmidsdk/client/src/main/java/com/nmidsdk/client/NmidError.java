package com.nmidsdk.client;

/**
 * Error raised by the nmid client when the server returns PDT_ERROR,
 * PDT_CANT_DO ("have no job do") or PDT_RATELIMIT.
 * Mirrors the node err.code pattern.
 */
public class NmidError extends RuntimeException {
    private static final long serialVersionUID = 1L;

    public final String code;

    public NmidError(String code, String message) {
        super(message);
        this.code = code;
    }
}
