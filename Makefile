# Copyright 2010 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.


peg: peg.peg.go main.go
	go build -o $@

peg.peg.go: peg.peg bootstrap_peg
	./bootstrap_peg $<

bootstrap_peg: bootstrap/*.go
	cd bootstrap && go build
	bootstrap/bootstrap
	rm -f peg.peg.go
	go build -o $@
	rm -f bootstrap.peg.go

clean:
	rm -f bootstrap/bootstrap bootstrap_peg peg peg.peg.go bootstrap.peg.go
