package shared

var Clientset, Dedicatedgameserverclientset = GetClientSet()
var secretsClient = Clientset.Core().Secrets(GameNamespace)
