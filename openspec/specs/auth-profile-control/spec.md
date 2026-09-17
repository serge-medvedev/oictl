## Purpose
Define authentication, profile, session, API key, and auth configuration commands for Open WebUI.

## Requirements

### Requirement: Session inspection
The CLI SHALL provide a command to inspect the current authenticated Open WebUI session.

#### Scenario: Current session is displayed
- **WHEN** the user runs `oictl auth me`
- **THEN** the CLI calls the auth session endpoint and prints the current session user information

### Requirement: Sign in and sign out
The CLI SHALL provide commands to sign in with supported credential payloads and sign out of the current session.

#### Scenario: User signs in
- **WHEN** the user runs `oictl auth login --file credentials.json`
- **THEN** the CLI submits the credentials to Open WebUI and prints the returned session data

#### Scenario: User signs out
- **WHEN** the user runs `oictl auth logout`
- **THEN** the CLI calls the signout endpoint and clears any persisted token for the selected profile when applicable

### Requirement: Profile updates
The CLI SHALL provide commands to update profile fields and timezone for the current user.

#### Scenario: Profile is updated
- **WHEN** the user runs `oictl auth profile update --file profile.json`
- **THEN** the CLI submits the profile update payload and prints the updated profile response

#### Scenario: Timezone is updated
- **WHEN** the user runs `oictl auth timezone set America/New_York`
- **THEN** the CLI submits the timezone update request and prints the server result

### Requirement: Password update
The CLI SHALL provide a command to update the current user's password without persisting the old or new password.

#### Scenario: Password is updated
- **WHEN** the user runs `oictl auth password update --file password.json`
- **THEN** the CLI submits the password update payload and does not write password values to profile storage

### Requirement: API key management
The CLI SHALL provide commands to create, fetch, and delete the current user's Open WebUI API key.

#### Scenario: API key is created
- **WHEN** the user runs `oictl auth api-key create`
- **THEN** the CLI calls the API key creation endpoint and prints the returned API key response

#### Scenario: API key is deleted
- **WHEN** the user runs `oictl auth api-key delete`
- **THEN** the CLI calls the API key deletion endpoint and reports the boolean server result

### Requirement: Auth admin configuration basics
The CLI SHALL provide admin commands to inspect and update auth configuration, LDAP configuration, LDAP server configuration, and OAuth configuration using Open WebUI auth endpoints.

#### Scenario: Auth admin config is read
- **WHEN** the user runs `oictl auth admin-config get`
- **THEN** the CLI prints the auth admin configuration returned by Open WebUI

#### Scenario: OAuth auth config is updated
- **WHEN** the user runs `oictl auth oauth-config set --file oauth.json`
- **THEN** the CLI submits the OAuth configuration payload and prints the updated response

### Requirement: Auth command authorization
The CLI SHALL surface Open WebUI authentication and authorization errors for auth/profile commands.

#### Scenario: Invalid credentials are rejected
- **WHEN** Open WebUI rejects a sign-in or protected auth request
- **THEN** the CLI exits non-zero and prints the response status and detail
