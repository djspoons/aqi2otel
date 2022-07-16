
.PHONY:
zip: cloud-function.zip

cloud-function.zip: *.go go.mod
	zip cloud-function.zip *.go go.mod
