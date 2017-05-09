# kvstore - a kvstore over http

kvstore provides key-value storage over HTTP.

Usage:

```bash
kvstore [FLAG]...
```

__Do NOT use to store your secrets.__


# Basic Usage

## Launch kvstore server

Create a json file called creds.json containing a list of accepted credentials/keys
that clients will use when getting or setting values.

```sh
# create credentials file
echo "[\"credential1\",\"cred2\"]" >creds.json

# launches the kvstore store listening on port 8080
kvstore 

# launch store on localhost (local kvstore)
kvstore -listen localhost:8080  

# use a diffent creds.json file
kvstore -creds ~/.kvcreds.json
```

The launched keystore can be used __via the command line__ or by `PUT` and `GET` __HTTP requests__.

## Use kvstore via command line

A client needs to set the environment variables 
   - `$KVSTORE` with the store endpoint
   - `$KVCRED` with one of the creds in `creds.json`

Use the `-set` and `get` flags to set and retrieve values.

``` sh
# export store server endpoint and cred
export KVSTORE=http://localhost:8080
export KVCRED=credential1

# set key value hello=world
kvstore -set -k "hello" -v "world"

# get the value for key hello
kvstore -get -k "hello"
```


## Use kvstore via HTTP requests

Setting key-value: 
Use `PUT` request with url form/query values:
   - `cred`: one of the values in `creds.json`
   - `k`: the key to be set
   - `v`: the value to be set

URL example: __PUT__ `http://localhost:8080?cred=credential1&k=hello&v=world`


Retrieving a value:
Use `GET` request with the following 
   - `cred`: one of the values in `creds.json`
   - `k`: the key whose value you want

URL example: __GET__ `http://localhost:8080?cred=credential1&k=hello&v=world`



