# Dlock

![Build](https://github.com/alexandreLamarre/dlock/actions/workflows/ci.yaml/badge.svg)
![Release](https://img.shields.io/github/v/release/alexandreLamarre/dlock)
[![License](https://img.shields.io/github/license/alexandreLamarre/dlock)](./LICENSE)

Dlock is a server for distributed locking for process synchronization & exclusive transactions in distributed systems.

Generally, a distributed lock can be used to coordinate access to a resource or interest in
such a resource in a distributed environment. In other words, distributed locks are useful when multiple systems need to reach a consensus about a shared resource.

## Lock Modes

As apposed to standard single-host locking mechanisms, distributed locks can be categorized into distinct modes with varying degress of distributed access guarantees.

- Null (NL) : indicates interest in the resource, but does not prevent other process from locking it
- Concurrent Read (CR) : Indicates a desire to read (but not update) the resources. Allows other processes to read or update the resource but prevents exclusive access to it.
- Concurrent Write (CW) : Indicates a desire to read and update the resource. It allows other processes to read or update the resource, but prevents others from having EX access to it.
- Protected Read (PR) : Traditional share lock, which indicates a desire to read the resource but prevents others from updating it. Others can however also read the resource.
- Protected Write (PW) : Traditional update lock, indicates a desire to read and update the resource and prevents others from updating it. Others with Concurrent read access can however read the resource
- Exclusive (EX) : Traditional exclusive lock, which allows read and update access to the resource, and prevents others from having access to it.

<table class="wikitable">

<tbody><tr>
<th>Mode</th>
<th>NL</th>
<th>CR</th>
<th>CW</th>
<th>PR</th>
<th>PW</th>
<th>EX
</th></tr>
<tr>
<th>NL
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:
</td></tr>
<tr>
<th>CR
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:
</td></tr>
<tr>
<th>CW
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:
</td></tr>
<tr>
<th>PR
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:
</td></tr>
<tr>
<th>PW
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:
</td></tr>
<tr>
<th>EX
</th>
<td style="background:#087324;vertical-align:middle;text-align:center;" class="table-yes">:white_check_mark:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:</td>
<td style="background:#ab0330;vertical-align:middle;text-align:center;" class="table-no">:x:
</td></tr></tbody></table>

### Notes

- A typical mutex corresponds one-to-one with a distributed exclusive lock (EX).
- There is no one-to-one mapping for a RW mutex in a distributed setting, although Protected Read & Write locks correspond closely to many of a RW mutexe's guarantees

## Support Matrix

|                    Backend / Lock Type                    |         EX         | PW  | PR  | CW  | CR  | NL  |
| :-------------------------------------------------------: | :----------------: | :-: | :-: | :-: | :-: | :-: |
| [Jetstream](https://docs.nats.io/nats-concepts/jetstream) | :white_check_mark: | :x: | :x: | :x: | :x: | :x: |
|                 [Etcd ](https://etcd.io/)                 | :white_check_mark: | :x: | :x: | :x: | :x: | :x: |
|                [Redis ](https://redis.io/)                | :white_check_mark: | :x: | :x: | :x: | :x: | :x: |

## Dlock specific guarantees

### Exclusive Locks (EX)

- **Liveliness A** : A lock is always eventually released when the process holding it crashes or exits unexpectedly.

- **Liveliness B** : A lock is always eventually released when its backend store is unavailable.

- **Atomicity A** : No two processes or threads can hold the same lock at the same time.

- **Atomicity B** : Any call to unlock will always eventually release the lock

## References

- [Distributed Lock Manager](https://en.wikipedia.org/wiki/Distributed_lock_manager). (n.d.). In Wikipedia. Retrieved from https://en.wikipedia.org/wiki/Distributed_lock_manager
- Kleppmann, Martin. "Designing Data-Intensive Applications." (2019).
- [Redis redlock algorithm](https://redis.io/docs/manual/patterns/distributed-locks/) from https://redis.io/docs/manual/patterns/distributed-locks/
