package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

const (
	_  = 1 << (10 * iota)
	KB // 1024
	MB // 1048576
	GB // 1073741824
	TB // 1099511627776             (exceeds 1 << 32)
	PB // 1125899906842624
	EB // 1152921504606846976
	ZB // 1180591620717411303424    (exceeds 1 << 64)
	YB // 1208925819614629174706176
)

func fileSize(n float64) string {
	if n < KB {
		return fmt.Sprintf("%fB", n)
	}
	if n < MB {
		return fmt.Sprintf("%fKB", float64(n)/(KB))
	}
	if n < GB {
		return fmt.Sprintf("%fMB", float64(n)/(MB))
	}
	return fmt.Sprintf("%fGB", float64(n)/(GB))
}

type goSize struct {
	total int
	kv    map[string]int
}

type Pair struct {
	Key   string
	Value int32
}

func (m *goSize) add(key string, size int) {
	e, ok := m.kv[key]
	if ok {
		e += size
		m.kv[key] = e
		return
	}
	m.kv[key] = size
}

func (m *goSize) format(top int) {
	var sizes []Pair
	for k, v := range m.kv {
		m.total += v
		sizes = append(sizes, Pair{k, int32(v)})
	}
	sort.Slice(sizes, func(i, j int) bool {
		return sizes[i].Value > sizes[j].Value
	})

	fmt.Printf("total: %s\n", fileSize(float64(m.total)))
	if top < 0 {
		for i := range sizes {
			if sizes[i].Key == "" {
				continue
			}
			fmt.Printf("size: %s\t\tpkg: %s\n", fileSize(float64(sizes[i].Value)), sizes[i].Key)
		}
		return
	}
	for i := 0; i < top; i++ {
		if sizes[i].Key == "" {
			continue
		}
		fmt.Printf("size: %s\t\tpkg: %s\n", fileSize(float64(sizes[i].Value)), sizes[i].Key)
	}
}

// T   text (code) segment symbol
// t   static text segment symbol
// R   read-only data segment symbol
// r   static read-only data segment symbol
// D   data segment symbol
// d   static data segment symbol
// B   bss segment symbol
// b   static bss segment symbol
// C   constant address
// U   referenced but undefined symbol
func nmTool(file string, lineChan chan string) error {
	cmd := exec.Command("go", "tool", "nm", "-size", file)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln("stdout pipe error: ", err)
	}

	log.Println("start ", cmd.Args)
	err = cmd.Start()
	if err != nil {
		log.Fatalln("start cmd error: ", err)
	}

	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString(byte('\n'))
			if err != nil || io.EOF == err {
				log.Println("cmd end")
				stdout.Close()
				close(lineChan)
				break
			}
			// log.Println("stdout: ", line)
			if strings.Contains(line, " U ") {
				continue
			}
			lineChan <- line
		}
	}()

	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func keygen(pkg string, level int) string {
	if strings.Contains(pkg, "/") {
		last := strings.LastIndex(pkg, "/")
		return pkg[:last]
	} else if strings.Contains(pkg, ".") {
		first := strings.Index(pkg, ".")
		return pkg[:first]
	}
	return pkg
}

func split(line string) []string {
	ss := strings.Split(line, " ")
	var nm []string
	for i := range ss {
		if ss[i] != "" {
			nm = append(nm, ss[i])
		}
	}
	return nm
}

func handle(lineCh chan string, level int) *goSize {
	m := &goSize{kv: make(map[string]int)}
	for {
		line, ok := <-lineCh
		if !ok {
			return m
		}

		// log.Println("recv: ", line)
		ss := split(line)
		if len(ss) > 3 {
			key := keygen(ss[3], level)
			size, err := strconv.Atoi(ss[1])
			if err != nil {
				log.Println(err)
			}
			m.add(key, size)
		}
	}
}

// gosize -f xxx -top 10
func main() {
	var file = flag.String("f", "", "file path")
	var top = flag.Int("top", 20, "top size numbers")
	var level = flag.Int("l", 2, "package level")
	flag.Parse()

	if *file == "" {
		log.Fatalln("file path is empty")
	}

	done := make(chan struct{})
	lineCh := make(chan string, 1<<12)
	go func() {
		s := handle(lineCh, *level)
		s.format(*top)
		done <- struct{}{}
	}()

	err := nmTool(*file, lineCh)
	if err != nil {
		log.Fatalln("go tool nm error: ", err)
	}

	<-done
}
