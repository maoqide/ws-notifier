package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
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

	group := strings.Trim(r.RequestURI, "/")
	// should be random generated
	sessionID := "123456"

	groupID := prefix + group
	n.Notify(groupID, tickerWorker, time.Hour*24)
	n.HandleRequestWithKeys(w, r, map[string]interface{}{"group": groupID, "id": groupID + "_" + sessionID})
	return
}

func tickerWorker(groupID string, sigChan chan int8, n *notifier.Notifier) error {
	worker := fmt.Sprintf("ticker_worker_%s_%d", groupID, time.Now().Unix())
	fmt.Printf("worker: %s\n", worker)

	defer func() {
		select {
		case sigChan <- 0:
			log.Printf("ticker worker: %s exit", worker)
		case <-time.After(time.Second * 3):
			log.Printf("ticker worker: %s exit after 3s delaying", worker)
		}
	}()
	ticker := time.NewTicker(time.Second * 2)
	count := 0
	for {
		fmt.Println(count)
		select {
		case signal := <-sigChan:
			log.Printf("receice stop signal %d for ticker worker: %s", signal, worker)
			return nil
		case <-ticker.C:
			err := n.GroupBroadcast([]byte(fmt.Sprintf("%s: %d", groupID, count)), groupID)
			if err != nil {
				log.Printf("err: %v", err)
			}
		}
		count++
	}
}
