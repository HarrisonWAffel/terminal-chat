# Terminal Chat

a Peer to peer terminal chat application featuring end to end AES encryption, a GUI, and other features.

![example](./screen-shot.png)

### Architecture
```
                                                    |          Steps
               --------------------                 |  1. Peer One POSTS token
       +------>| Discovery Server | <------+        |     and webRTC info, waits 
       |       --------------------        |        |     
       |                                   |        |  2. DS Stores webRTC   
       v                                   v        |     info using token
 ------------                        ------------   |              
 | Peer One | <--------------------> | Peer Two |   |  3. Peer Two POSTS webRTC 
 ------------                        ------------   |     info and gives token
                                                    |   
                                                    |  4. DS forwards P2 webRTC
                                                    |     info to P1 and P1 
                                                    |     webRTC info to P2.
                                                    | 
                                                    |  5. Clients directly connect 
                                                    |     to one another  
```                        


### How to Run 

This application needs to connect to a dedicated server so that peers may share their
connection details with one another using custom tokens. The dedicated server only
holds onto connection details for ten minutes at most, and deletes each token once
both parties have connected to one another. A Dockerfile for the server has been provided. 
This server must be exposed to the internet if you wish to have peers from outside of 
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
./terminal-chat -server-url=${SERVER_URL} -screen-name=host -create
```

To connect to a conversation run the following command 
```bash 
./terminal-chat -server-url=${SERVER_URL} -screen-name=guest -connect=${TOKEN}
```

If you do not want to provide the `SERVER_URL` each time you run the command you can set 
the `TERMINAL_CHAT_URL` environment variable equal to the server URL.