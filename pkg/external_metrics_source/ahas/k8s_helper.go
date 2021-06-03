package ahas

import (
	"errors"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "k8s.io/klog"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const K8S_NAMESPACE_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

const AHAS_APP_NAME_ANNOTATION_KEY = "ahasAppName"
const AHAS_NAMESPACE_ANNOTATION_KEY = "ahasNamespace"

var k8sClient = getK8sClient()

type SentinelPilotMetadata struct {
	appName   string
	namespace string
}

// get a clientset with in-cluster config.
func getK8sClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err)
		return nil
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
	}
	return clientset
}

func getCurrentK8sNamespace() (string, error) {
	f, err := os.Open(K8S_NAMESPACE_PATH)
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func getPilotAnnotationMetadata(namespace string) (*SentinelPilotMetadata, error) {
	println(len(namespace))
	if len(namespace) == 0 {
		return nil, errors.New("invalid namespace")
	}
	// log.Infoln("Namespace resolved:" + namespace)
	ds, err := k8sClient.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// log.Infof("Deployment resolved, number: %d\n", len(ds.Items))
	pilotMetadata := &SentinelPilotMetadata{}
	for _, deployment := range ds.Items {
		if len(pilotMetadata.appName) > 0 {
			break
		}
		metaAnnotations := deployment.Spec.Template.ObjectMeta.Annotations
		for k, v := range metaAnnotations {
			// log.Infof("Annotation resolved, deployment: %s, k: %s, v: %s\n", deployment.Name, k, v)
			if k == AHAS_APP_NAME_ANNOTATION_KEY {
				pilotMetadata.appName = v
			} else if k == AHAS_NAMESPACE_ANNOTATION_KEY {
				pilotMetadata.namespace = v
			}
		}
	}
	return pilotMetadata, nil
}
