# Fault Tolerance in Distributed Systems for Deep Learning

Describe techniques for achieving fault tolerance in
distributed systems designed mainly for Deep Learning. Explain how redundancy,
replication, and consensus algorithms contribute to fault tolerance, providing
examples where applicable.

## answer

- Fault tolerance is important, especially in deep learning where long running processes being interrupted would be very annoying
and potentially very bad. Ideally a process should be able to continue even if nodes fail and be able to recover data from failed
notes as well as be able to detect when nodes fail. There are a variety of distributed systems techniques to conver all bases
each with their tradeoffs. 

- One of the most important approaches is redundancy, where sytems like Tensorflow and PyTorch implement data parallelism, and thus
have multiple copies of the model across worker nodes. This way if any data is lost it is almost always available elsewhere. This was also
part of the DynamoDB assignment where we would put data on multiple different for redundancy. Another key aspect is replication. 
By having checkpointing mechanism, like terms in raft, we can keep a log and store existing progress. For example in deep learning, 
we can store model weights and optimizer states and training progress in storage systems like AWS S3 so that way training can resume
from the most recent available checkpoint. This way we can avoid having to start over even if every single redundant node fails. 

- Another really important aspect are consensus algorithms. 

- Consensus are also really important for fault tolerance. They ensure that all nodes in the system agree on the current state despite potential failures. For example, parameter servers in distributed training often use protocols like Paxos or Raft to make sure model updates are consistently applied across all workers. In real-world frameworks, we see this implemented through different training approaches. Asynchronous strategies like those in Horovod allow training to continue even when some nodes fail, while synchronous approaches might halt the entire process if even one node goes down. Systems also use simple but effective fault detection through heartbeats and timeouts to quickly identify failed nodes.  By combining redundancy, replication, and consensus mechanisms, distributed deep learning systems can continue operating smoothly even when parts of the infrastructure fail.
