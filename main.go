package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	endpoint = "http://smb.cidb.gov.my/contractor/contractors/information/%d"
	min      = 1
	max      = 217898
)

var (
	errNotOk = errors.New("Not status 200")
)

func checkerr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func fileprefix(prefix, path string) string {
	return fmt.Sprintf("%s/%s", prefix, path)
}

func main() {
	var threadCount int
	var retry string
	flag.IntVar(&threadCount, "thread", 10, "thread count for http requests")
	flag.StringVar(&retry, "retry", "", "retry file")
	flag.Parse()

	now := time.Now()
	prefix := fmt.Sprintf("summary_%d%02d%02d_%02d%02d",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(),
	)

	err := os.MkdirAll(prefix, 0777)
	checkerr(err)

	donefile, err := os.Create(fileprefix(prefix, "done.log"))
	checkerr(err)
	donelog := log.New(donefile, "", 0)

	errorfile, err := os.Create(fileprefix(prefix, "error.log"))
	checkerr(err)
	errorlog := log.New(errorfile, "", 0)

	retryfile, err := os.Create(fileprefix(prefix, "retry.log"))
	checkerr(err)
	retrylog := log.New(retryfile, "", 0)

	err = os.MkdirAll("downloaded", 0777)
	checkerr(err)

	var wg sync.WaitGroup
	c := make(chan string)

	for x := 0; x < threadCount; x++ {
		wg.Add(1)
		go func() {
			client := &http.Client{}
			for {
				target, next := <-c
				if !next {
					break
				}

				if err := do(client, target); err != nil {
					errorlog.Printf("%v (target: %s)", err, target)
					retrylog.Println(target)
				} else {
					donelog.Println(target)
				}

				<-time.After(time.Duration(rand.Intn(10)) * time.Second)
			}
			wg.Done()
		}()
	}

	if retry != "" {
		retryFile, err := os.Open(retry)
		checkerr(err)

		scanner := bufio.NewScanner(retryFile)
		for scanner.Scan() {
			c <- scanner.Text()
		}
	} else {
		for i := min; i < max+1; i++ {
			target := fmt.Sprintf(endpoint, i)
			c <- target
		}
	}
	close(c)

	wg.Wait()
}

func do(client *http.Client, target string) error {
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", giveUserAgents())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errNotOk
	}

	parts := strings.Split(target, "/")
	filename := fmt.Sprintf("downloaded/%s", parts[len(parts)-1])
	dest, err := os.Create(filename)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dest, resp.Body); err != nil {
		return err
	}

	return nil
}

var fakeUserAgents = []string{
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:40.0) Gecko/20100101 Firefox/40.1",
	"Mozilla/5.0 (Windows NT 6.3; rv:36.0) Gecko/20100101 Firefox/36.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10; rv:33.0) Gecko/20100101 Firefox/33.0",
	"Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2227.1 Safari/537.36",
	"Mozilla/5.0 (compatible, MSIE 11, Windows NT 6.3; Trident/7.0; rv:11.0) like Gecko",
	"Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1; Trident/6.0)",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.75.14 (KHTML, like Gecko) Version/7.0.3 Safari/7046A194A",
}

func giveUserAgents() string {
	return fakeUserAgents[rand.Intn(len(fakeUserAgents))]
}
