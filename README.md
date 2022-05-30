# erpc
a rpc framework base event-loop

erpc is tcp rpc framework. It is highly inspired by [muduo](https://github.com/chenshuo/muduo).

In general, client will monopolize a connection to send request until receive resposne or timeout, the connection do nothing while waiting response. 

In high concurrency scenarios, many connections will be created. If response time of connection too long, it maybe make os handle too many connection at the same.

We can do something while tcp connection waiting response. erpc use a non-block IO send message, and callback to handle response.