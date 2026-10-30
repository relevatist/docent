//! The docent Zellij plugin. docent sends each pane write as a pipe message.
//! Unlike `zellij action write`, `write_to_pane_id` does not clear the scroll
//! of the caller's focused pane.
//!
//! | Pipe name      | Writes                                    |
//! |----------------|-------------------------------------------|
//! | `docent-write` | the payload                               |
//! | `docent-paste` | the payload as a bracketed paste          |

use std::collections::BTreeMap;
use std::time::{SystemTime, UNIX_EPOCH};
use zellij_tile::prelude::*;

// Zellij drops these from a write to a pane without bracketed paste mode
// only when the write is exactly the marker, so each one is its own write.
const PASTE_BEGIN: &[u8] = b"\x1b[200~";
const PASTE_END: &[u8] = b"\x1b[201~";

struct Write {
    pipe: String,
    pane: PaneId,
    deadline: u128,
    writes: Vec<Vec<u8>>,
}

#[derive(Default)]
struct State {
    client_id: ClientId,
    granted: bool,
    pending: Vec<Write>,
}

register_plugin!(State);

impl ZellijPlugin for State {
    fn load(&mut self, _: BTreeMap<String, String>) {
        request_permission(&[
            PermissionType::WriteToStdin,
            PermissionType::ReadCliPipes,
            PermissionType::ReadApplicationState,
        ]);
        subscribe(&[EventType::ListClients, EventType::PermissionRequestResult]);
        self.client_id = get_plugin_ids().client_id;
    }

    // Each attached client runs its own instance and every instance gets the
    // message. The client list comes back later as an event, so each
    // instance holds the pipe open until then. Only the instance of the
    // lowest client writes, answers, and releases the pipe.
    //
    // ponytail: when the lowest client detaches between list_clients and
    // the event, two instances can both see themselves as lowest and write
    // twice. This race is accepted; docent reads any "ok" as written, so it
    // does not write a third time.
    //
    // Without permissions the plugin does not hold the pipe or answer, so
    // docent falls back at once.
    fn pipe(&mut self, msg: PipeMessage) -> bool {
        let PipeSource::Cli(pipe) = msg.source else {
            return false;
        };
        // The CLI sends one more message with no payload after the release,
        // because its stdin is not a terminal.
        let Some(payload) = msg.payload else {
            return false;
        };
        let writes = match msg.name.as_str() {
            "docent-write" => vec![payload.into_bytes()],
            "docent-paste" => vec![
                PASTE_BEGIN.to_vec(),
                payload.into_bytes(),
                PASTE_END.to_vec(),
            ],
            _ => return false,
        };
        let Some(Ok(pane)) = msg.args.get("pane").map(|p| p.parse()) else {
            return false;
        };
        let Some(Ok(deadline)) = msg.args.get("deadline").map(|d| d.parse()) else {
            return false;
        };
        if !self.granted {
            return false;
        }
        block_cli_pipe_input(&pipe);
        self.pending.push(Write {
            pipe,
            pane,
            deadline,
            writes,
        });
        list_clients();
        false
    }

    fn update(&mut self, event: Event) -> bool {
        let clients = match event {
            Event::PermissionRequestResult(status) => {
                self.granted = status == PermissionStatus::Granted;
                return false;
            }
            Event::ListClients(clients) => clients,
            _ => return false,
        };
        let lowest = clients.iter().map(|c| c.client_id).min();
        for w in self.pending.drain(..) {
            if lowest != Some(self.client_id) {
                continue;
            }
            // After the deadline docent may have fallen back already.
            if now_ms() > w.deadline {
                cli_pipe_output(&w.pipe, "late");
            } else {
                for b in w.writes {
                    write_to_pane_id(b, w.pane);
                }
                cli_pipe_output(&w.pipe, "ok");
            }
            unblock_cli_pipe_input(&w.pipe);
        }
        false
    }
}

fn now_ms() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| d.as_millis())
}
