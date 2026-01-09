# The watch-objs command

This is a simple example client of the generated code.

This client focuses on one Kubernetes namespace and makes an informer on all the
KubeFlex objects in that namespace.
When informed of any add/delete/update of such an object, a line is logged;
V(2) for add/delete, V(4) for update.
