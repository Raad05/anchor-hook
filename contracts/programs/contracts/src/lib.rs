use anchor_lang::prelude::*;

declare_id!("DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD");

// ---------------------------------------------------------------------------
// Event definition
// ---------------------------------------------------------------------------

/// Emitted by `trigger_action`. The Go indexer will decode this from the
/// base64-encoded `Program data: …` log line.
#[event]
pub struct UserAction {
    /// The wallet that triggered the action.
    pub user: Pubkey,
    /// A short label identifying what action was performed (e.g. "transfer",
    /// "stake", "vote").  Kept to ≤32 bytes on-chain to keep log overhead low.
    pub action_type: String,
    /// A generic numeric quantity associated with the action (lamports, tokens,
    /// vote weight, …).
    pub amount: u64,
}

// ---------------------------------------------------------------------------
// Program
// ---------------------------------------------------------------------------

#[program]
pub mod contracts {
    use super::*;

    /// Emit a `UserAction` event.  No accounts are mutated; the instruction
    /// exists purely to produce the log that the Go listener will consume.
    pub fn trigger_action(
        ctx: Context<TriggerAction>,
        action_type: String,
        amount: u64,
    ) -> Result<()> {
        emit!(UserAction {
            user: ctx.accounts.user.key(),
            action_type,
            amount,
        });
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Account context
// ---------------------------------------------------------------------------

#[derive(Accounts)]
pub struct TriggerAction<'info> {
    /// The signer whose public key is captured in the event.
    #[account(mut)]
    pub user: Signer<'info>,
}
