package defrag


// Defrag reads fragments of the packets from the connection 
// collects packets in a cache. When all packets of a frame are collected writes
// the whole frame to the output
// Cutting corners: 
//  * Assume that all fragments arrive, no timeout 
//  * No error checks 
//  * The RAM is unlimited 
func Defrag(connection net.PacketConn) io.Reader {

}