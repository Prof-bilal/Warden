# Cross-platform agent isolation

| Platform | Primary boundary | Read/write | Descendants | Fallback | Guarantee caveat |
| --- | --- | --- | --- | --- | --- |
| Linux | bubblewrap + Landlock | yes | namespaces and Landlock | Docker | kernel and user-namespace availability must be probed |
| macOS | Seatbelt profile | profile-dependent | inherited | Docker | `sandbox-exec` is deprecated and semantics vary by macOS release |
| Windows | AppContainer + DACL + Job Object + WFP | yes | token/job inheritance | Docker/unsupported | reparse points, ACL inheritance, and host edition capabilities matter |

Every backend must report capabilities before launch. A policy requiring a
guarantee unavailable on the host must fail closed rather than silently degrade.

Linux tests must cover mount aliases, Landlock inheritance, `/proc`, descriptor
inheritance, hard links, symlinks, and namespace process visibility. macOS tests must
cover Seatbelt path rules and descendant execution. Windows tests must cover junctions,
reparse points, case-insensitive aliases, UNC/device paths, ACL restoration, and Job
Object descendant cleanup.
