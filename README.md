# Terminal Chat

a Peer to peer terminal chat application featuring end to end AES encryption and other features. 

### Architecture
```
                                                    |          Steps
               --------------------                 |  1. Peer One POSTS token
       +------>| Discovery Server | <------+        |     and conn info, waits 
       |       --------------------        |        |     
       |                                   |        |  2. DS Creates Disocvery   
       v                                   v        |     token
 ------------                        ------------   |              
 | Peer One | <--------------------> | Peer Two |   |  3. Peer Two POSTS conn 
 ------------                        ------------   |     info and gives token
                                                    |   
                                                    |  4. DS forwards P2 conn
                                                    |     info to P1 and P2 
                                                    |     conn info to P1. 
```                        


### How to Run 

This application needs to connect to a dedicated server so that peers may share their
connection details with one another using custom tokens. The dedicated server only
holds onto connection details for ten minutes at most, and deletes each token once
both parties have connected to one another. A Dockerfile for the server has been provided,
however this server must be exposed to the internet if you wish to have peers from outside of 
your local network connect to you. To start the dedicated server, compile this repository and run 
the following command 

To run without the docker container
```bash
./terminal-chat -server -server-port=":8081"
```

To run with the docker container
```bash 
    docker build . -t terminal-chat-server && docker run -p 8081:8081 -d terminal-chat-server 
```

Once a dedicated server has been created peers can begin to connect with one another using the following commands

To create a new connection token use the following command, you can provide a custom token or use a generated UUID as a token
```bash 
    ./terminal-chat -server-url=${SERVER_URL} -create -screen-name=host
```

To connect to a conversation run the following command 
```bash 
    ./terminal-chat -server-url=${SERVER_URL} -screen-name=guest -connect=${TOKEN}
```

