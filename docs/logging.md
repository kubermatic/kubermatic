# Logging

We are using [glog](https://github.com/golang/glog) at the moment as logging library.

[glog](https://github.com/golang/glog) might be replaced with [klog](https://github.com/kubernetes/klog) in the future.
As klog follows the same interface as [glog](https://github.com/golang/glog), the following guidelines apply for both.


## Guidelines

We're following the [guidelines from Kubernetes](https://github.com/kubernetes/community/blob/b3349d5b1354df814b67bbdee6890477f3c250cb/contributors/devel/logging.md).


## Handling old code

We've not followed that guidelines in the past, thus we might encounter code which does not apply to the above mentioned guidelines.
This code should be converted as soon as it get's modified.
It is the responsibility of the individual PR author & reviewer to make sure, modified code parts adhere to the guidelines.
