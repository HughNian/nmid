package nmid.worker;

import java.io.*;
import java.io.OutputStream;
import java.io.IOException;
import java.net.*;
import java.net.Socket;
import java.net.SocketTimeoutException;
import nmid.consts.Constants;

/**
 * Agent class
 *
 */
public class Agent {
    public String net;
    public String addr;
    public Socket conn;
    public Worker worker;
    public Request req;
    public Response res;
    public int lastTime;

    public Agent(String net, String addr, Worker worker) {
        this.net = net;
        this.addr = addr;
        this.worker = worker;
    }

    public void Connect() throws Exception {
        try {
            // 解析地址和端口号
            String[] addrParts = this.addr.split(":");
            String ipAddress = addrParts[0];
            int port = Integer.parseInt(addrParts[1]);

            this.conn = new Socket();
            this.conn.connect(new InetSocketAddress(ipAddress, port), Constants.DIAL_TIME_OUT);

            // Start background work if needed
            new Thread(() -> Work()).start();
        } catch (Exception e) {
            throw e;
        }
    }

    public void Write() {
        byte[] buf = this.req.EncodePack();

        int totalBytesToWrite = buf.length;
        try {
            for (int offset = 0; offset < totalBytesToWrite;) {
                this.conn.getOutputStream().write(buf, offset, totalBytesToWrite - offset);
                offset += totalBytesToWrite - offset;
            }
        } catch (IOException e) {
            System.err.println("Error during write operation: " + e.getMessage());
        }
    }


    public void Work() {

    }
}
