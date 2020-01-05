package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	notifier "github.com/maoqide/ws-notifier"
)

func main() {
	fmt.Println("hello")

	http.HandleFunc("/", handleFunc)
	http.ListenAndServe(":8080", nil)
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	prefix := "ticker_"
	n := notifier.Default()

	group := r.RequestURI
	// should be random generated
	sessionID := "123456"

	groupID := prefix + group
	n.Notify(group, tickerWorker, time.Hour*24)
	n.HandleRequestWithKeys(w, r, map[string]interface{}{"group": groupID, "id": group + "_" + sessionID})
	return
}

func tickerWorker(groupID string, sigChan chan int8, n *notifier.Notifier) error {
	worker := fmt.Sprintf("ticker_worker_%d", time.Now().Unix())

	defer func() {
		select {
		case sigChan <- 0:
			log.Printf("ticker worker: %s exit", worker)
		case <-time.After(time.Second * 3):
			log.Printf("ticker worker: %s exit after 3s delaying", worker)
		}
	}()
	fmt.Println("--------")
	count := 0
	for {
		fmt.Println(count)
		select {
		case signal := <-sigChan:
			log.Printf("receice stop signal %d for ticker worker: %s", signal, worker)
			return nil
		default:
			time.Sleep(2)
			err := n.GroupBroadcast([]byte(fmt.Sprintf("%s: %d", groupID, count)), groupID)
			if err != nil {
				fmt.Printf("====%v", err)
			}
		}
		count++
	}
}
