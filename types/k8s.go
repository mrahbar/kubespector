package types

type ServicePort struct {
	Name       string
	Protocol   string
	Port       int
	TargetPort int
}

type PodPort struct {
	Name     string
	Protocol string
	Port     int
}

type Env struct {
	Name       string
	Value      string
	FieldValue string
}

type Arg struct {
	Key   string
	Value string
}

type ResourceRequest struct {
	Cpu    string
	Memory string
}

type Service struct {
	Name      string
	Namespace string
	Ports     []ServicePort
}

type ReplicationController struct {
	Name            string
	Namespace       string
	Image           string
	NodeName        string
	Args            []Arg
	Commands        []string
	Ports           []PodPort
	ResourceRequest ResourceRequest
	Envs            []Env
}

const (
	NAMESPACE_TEMPLATE = `apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
`
	SERVICE_TEMPLATE = `apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
  namespace: {{.Namespace}}
spec:
  ports:{{range $i, $a := .Ports}}
  - name: {{.Name}}
    protocol: {{.Protocol}}
    port: {{.Port}}
    targetPort: {{.TargetPort}}{{end}}
  selector:
    app: {{.Name}}
  type: ClusterIP
`

	REPLICATION_CONTROLLER_TEMPLATE = `apiVersion: v1
kind: ReplicationController
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  replicas: 1
  selector:
    app: {{.Name}}
  template:
    metadata:
      name: {{.Name}}
      labels:
        app: {{.Name}}
    spec:
    {{- if .NodeName }}
      nodeName: {{.NodeName}}
    {{- end}}
      containers:
      - name: {{.Name}}
        image: {{.Image}}
        imagePullPolicy: Always
	{{- if .Args }}
        args:
        {{- range $i, $a := .Args}}
        - {{.Key}}={{.Value}}
        {{- end}}
	{{- end}}
	{{- if .Commands }}
        command:
        {{- range $i, $e := .Commands}}
          - {{ $e }}
        {{- end}}
	{{- end}}
	{{- if .Ports }}
        ports:
        {{- range $i, $a := .Ports}}
        - name: {{.Name}}
          protocol: {{.Protocol}}
          containerPort: {{.Port}}
        {{- end}}
	{{- end}}
		{{- if .Envs }}
        env:
	{{- range $i, $a := .Envs}}
        - name: {{.Name}}
		  {{- if .FieldValue }}
          valueFrom:
            fieldRef:
              fieldPath: {{.FieldValue}}
          {{- else}}
          value: "{{.Value}}"
          {{- end}}
        {{- end}}
	{{- end}}
	{{- if .ResourceRequest }}
        resources:
		  requests:
		  {{- if .ResourceRequest.Cpu}}
		    cpu: {{.ResourceRequest.Cpu}}
          {{- end}}
		  {{- if .ResourceRequest.Memory}}
		    memory: {{.ResourceRequest.Memory}}
          {{- end}}
	{{- end}}
`
)
