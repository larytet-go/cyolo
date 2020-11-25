
Try

    go test -v ./...


Usage 

    import "defrag"

    func main() {
        reader := defrag.New(connection)
        buf := make([]byte, 1024)
        count, err := reader.Read(buf)
        // .....
    }
