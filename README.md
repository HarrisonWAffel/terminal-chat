# Terminal Chat

a Peer to peer terminal chat application featuring end to end encryption and other features. 



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

Peers post their connection info as a base64 string which is decoded 
into the required JSON / connection string. 