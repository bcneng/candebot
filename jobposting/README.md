Applicación para BcnEng Slack con el fin de simplificar la publicación de ofertas de trabajo. 



> La app estuvo desarrollada en Python 3 con el objectivo de ser de ser deployada en Google Functions. En caso de deployarse en un ambiente distinto, es probable que se requieran cambios.

---

Para confiugrarla en un Workspace de slack se requiere:

- Instalar la app y obtener token para utilizar Slack API. El token debe estar disponible como _envirorment variable_ : __SLACK_API_KEY__ a la hora de ejecutar el backend. 
- Conceder los siguientes permisos de OAuth 2:
    - chat:write
    - commands
    - incoming-webhook
    - users:read
- Agregar un slash command para llamar a la app desde un channel de slack (`/post-job` por ejemplo). Agregar la URL de la app deployada como endpoint.
- Configurar la request URL en _Interactive compoents_ igual a la utilizada en el slash command (La URL de la app deployada).