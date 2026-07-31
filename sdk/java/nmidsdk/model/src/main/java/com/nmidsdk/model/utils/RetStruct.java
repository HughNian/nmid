package com.nmidsdk.model.utils;

/**
 * Standard nmid return struct: { Code, Msg, Data }.
 * Mirrors the Go wor.GetRetStruct() / node buildRetStruct pattern.
 * Fields are public so callers can read them directly.
 */
public class RetStruct {
    public int Code;
    public String Msg;
    public byte[] Data;
}
