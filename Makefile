install:
	go install .

build:
	go build .

js:
	gogio -target js .

ios:
	gogio -target ios -appid us.sailrs.arrows .

clean:
	-rm -rf arrows arrows.app arrows.ipa arrows.apk
