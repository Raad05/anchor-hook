use anchor_lang::prelude::*;

declare_id!("DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD");

#[program]
pub mod contracts {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        msg!("Greetings from: {:?}", ctx.program_id);
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize {}
