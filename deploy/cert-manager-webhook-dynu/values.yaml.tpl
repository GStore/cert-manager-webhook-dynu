# The GroupName here is used to identify your company or business unit that
# created this webhook.
# For example, this may be "acme.mycompany.com".
# This name will need to be referenced in each Issuer's `webhook` stanza to
# inform cert-manager of where to send ChallengePayload resources in order to
# solve the DNS01 challenge.
# This group name should be **unique**, hence using your own company's domain
# here is recommended.
groupName: gunstore.github.com

deployment:
  loglevel: 6

certManager:
  namespace: cert-manager
  serviceAccountName: cert-manager

image:
  repository: {IMAGE_NAME}
  tag: {IMAGE_TAG}
  pullPolicy: IfNotPresent

credentialsSecretRef: dynu-credentials

nameOverride: ""
fullnameOverride: ""

service:
  type: ClusterIP
  port: 443

resources: {}
  # We usually recommend not to specify default resources and to leave this as a conscious
  # choice for the user. This also increases chances charts run on environments with little
  # resources, such as Minikube. If you do want to specify resources, uncomment the following
  # lines, adjust them as necessary, and remove the curly braces after 'resources:'.
  # limits:
  #  cpu: 100m
  #  memory: 128Mi
  # requests:
  #  cpu: 100m
  #  memory: 128Mi

nodeSelector: {}
tolerations: []

affinity: {}
