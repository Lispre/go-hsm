package main

import "fmt"
import hsm "github.com/hhkbp2/go-hsm"

func main() {
    m, _ := hsm.NewWorld()
    events := []hsm.Event{
        &hsm.StdEvent{hsm.Event2},
        &hsm.StdEvent{hsm.Event1},
        &hsm.StdEvent{hsm.Event1},
        &hsm.StdEvent{hsm.Event2},
    }
    for _, e := range events {
        fmt.Println("> dispatch event:", e)
        m.Dispatch(e)
    }
}
