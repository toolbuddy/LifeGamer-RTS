# BUG
- Map data update in notifier will have multiple goroutine running on multi user login, this causes performance problem.

# TODO
- Prevent WAW problem on player & map DB
