# ClipBridge Windows

ClipBridge Windows is planned as a lightweight clipboard sync agent for Windows.

It should not replace the native Windows clipboard history UI. Instead, it will:

- listen for clipboard changes while the agent is running,
- filter and deduplicate supported clipboard records,
- encrypt records locally,
- upload encrypted records to ClipBridge Server,
- optionally pull recent cloud records and copy selected items back into the
  Windows clipboard.

The first implementation target is text-only sync. Image, HTML, RTF, and file
support can be added after the protocol and security model are stable.

## Proposed Stack

- C#/.NET for tray integration and Windows clipboard APIs
- DPAPI or Windows Credential Manager for local secret protection
- Shared protocol documents from the ClipBridge docs folder

