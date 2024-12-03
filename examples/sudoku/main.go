package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sjnam/dlx"
	"github.com/sjnam/dlx/util"
)

func sudokuDLX(rd io.Reader) io.Reader {
	var c, j int
	var pos, row, col, box [9][9]int
	buf := make([]byte, 9)

	for {
		if _, err := io.ReadFull(rd, buf); err == io.EOF {
			break
		}

		for k := 0; k < 9; k++ {
			if buf[k] >= '1' && buf[k] <= '9' {
				d := int(buf[k] - '1')
				x := j/3*3 + k/3
				pos[j][k] = d + 1
				if row[j][d] != 0 {
					log.Fatalf("digit %d appears in columns"+
						" %d and %d of row %d!", d+1, row[j][d]-1, k, j)
				}
				row[j][d] = k + 1
				if col[k][d] != 0 {
					log.Fatalf("digit %d appears in rows"+
						" %d and %d of column %d!", d+1, col[k][d]-1, j, k)
				}
				col[k][d] = j + 1
				if box[x][d] != 0 {
					log.Fatalf("digit %d appears in rows"+
						" %d and %d of box %d!", d+1, box[x][d]-1, j, x)
				}
				box[x][d] = j + 1
				c++
			}
		}
		j++
	}

	r, w := io.Pipe()
	go func() {
		defer func() {
			_ = w.Close()
		}()

		for j = 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				if pos[j][k] == 0 {
					fmt.Fprintf(w, "p%d%d ", j, k)
				}
			}
		}
		for j = 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				if row[j][k] == 0 {
					fmt.Fprintf(w, "r%d%d ", j, k+1)
				}
			}
		}
		for j = 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				if col[j][k] == 0 {
					fmt.Fprintf(w, "c%d%d ", j, k+1)
				}
			}
		}
		for j = 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				if box[j][k] == 0 {
					fmt.Fprintf(w, "b%d%d ", j, k+1)
				}
			}
		}
		fmt.Fprintln(w)

		for j = 0; j < 9; j++ {
			for k := 0; k < 9; k++ {
				for d := 0; d < 9; d++ {
					x := j/3*3 + k/3
					if pos[j][k] == 0 && row[j][d] == 0 &&
						col[k][d] == 0 && box[x][d] == 0 {
						fmt.Fprintf(w, "p%d%d r%d%d c%d%d b%d%d\n",
							j, k, j, d+1, k, d+1, x, d+1)
					}
				}
			}
		}
	}()

	return r
}

func danceSudoku(line interface{}) interface{} {
	lstr := line.(string)
	xc := dlx.NewDancer()
	res := xc.Dance(sudokuDLX(strings.NewReader(lstr)))
	ans := []byte(lstr)
	for _, opt := range <-res.Solutions {
		x := int(opt[0][1] - '0')
		y := int(opt[0][2] - '0')
		ans[x*9+y] = opt[1][2]
	}
	return [][]byte{[]byte(lstr), ans}
}

func inputLines(fd io.Reader) <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			ch <- strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()

	return ch
}

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Fatalf("usage: %s file\n", args[0])
	}

	start := time.Now()

	fd, err := os.Open(args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	toSquare := func(
		ctx context.Context,
		valueStream <-chan interface{},
	) <-chan [][]byte {
		stringStream := make(chan [][]byte)
		go func() {
			defer close(stringStream)
			for v := range valueStream {
				select {
				case <-ctx.Done():
					return
				case stringStream <- v.([][]byte):
				}
			}
		}()
		return stringStream
	}

	i := 0
	for s := range toSquare(ctx, util.OrderedProcess(ctx, inputLines(fd), danceSudoku, 8)) {
		i++
		fmt.Printf("Q[%5d]: %s\n", i, s[0])
		fmt.Printf("A[%5d]: %s\n", i, s[1])
	}

	fmt.Printf("Solving took: %v\n", time.Since(start))
}
