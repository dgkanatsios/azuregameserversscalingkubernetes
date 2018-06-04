# Resources

Some useful resources regarding Kubernetes Operators, CRDs, etc.

- Kubernetes sample controller: https://github.com/kubernetes/sample-controller and https://github.com/kubernetes/client-go/tree/master/examples/workqueue
- Joe Beda's TGIK sample repo and YouTube videos: https://github.com/jbeda/tgik-controller
- Thomas Stringer's blog post: https://medium.com/@trstringer/create-kubernetes-controllers-for-core-and-custom-resources-62fc35ad64a3
- Kube-controller-demo by Aaron Levy: https://github.com/aaronlevy/kube-controller-demo
- Analyzing value of Operator Framework for Kubernetes community: https://itnext.io/analyzing-value-of-operator-framework-for-kubernetes-community-5a65abc259ec
- Steps to generate CRD/Operator code: https://github.com/cloud-ark/kubeplus/issues/14
- kubebuilder: https://github.com/kubernetes-sigs/kubebuilder
- CoreOS operator framework: https://coreos.com/blog/introducing-operator-framework
- A thread on Reddit: https://www.reddit.com/r/kubernetes/comments/8ien90/if_i_were_to_build_an_operator_what_should_i_use/
- Heptio blog post: https://blog.heptio.com/an-introduction-to-extending-kubernetes-with-customresourcedefinitions-76deb675b27a
- Openshift CRD Deep Dive: https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

Interesting blog posts from Cloudark
- https://medium.com/@cloudark/kubernetes-custom-controllers-b6c7d0668fdf
- https://medium.com/@cloudark/why-to-write-kubernetes-operators-9b1e32a24814
- https://itnext.io/analyzing-value-of-operator-framework-for-kubernetes-community-5a65abc259ec
- https://itnext.io/under-the-hood-of-kubebuilder-framework-ff6b38c10796
- https://medium.com/@cloudark/under-the-hood-of-the-operator-sdk-eebc8fdeebbf

Other stuff

- Couldn't get dep to work with client-go, so we used these [simple scripts](https://github.com/ahmetb/dotfiles/tree/master/bin) from [@ahmetb](https://github.com/ahmetb) to vendor the dependencies

