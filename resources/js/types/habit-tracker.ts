export type HtFlash = {
    success?: string | null;
    error?: string | null;
    achievement?: HtAchievement | null;
};

export type HtAchievement = {
    person_id: number;
    reward_id: number;
    person_name: string;
    reward_name: string;
};

export type HtPerson = {
    id: number;
    name: string;
    display_order: number;
};

export type HtPersonWithTrash = HtPerson & {
    deleted_at: string | null;
};

export type HtReward = {
    id: number;
    name: string;
    cost: number;
    achieved_at: string | null;
};

export type HtHabit = {
    id: number;
    name: string;
    display_order: number;
    tick_dates: string[];
};

export type HtSprintDay = {
    date: string;
    is_weekend: boolean;
    is_month_start: boolean;
};

export type HtSprintMeta = {
    id: number;
    start_date: string;
    end_date: string;
    status: 'active' | 'archived';
};

export type HtParticipant = {
    id: number;
    person: HtPerson;
    display_order: number;
    carry_forward_balance: number;
    points_balance: number;
    active_reward: HtReward | null;
    habits: HtHabit[];
};

export type HtGrid = {
    sprint: HtSprintMeta;
    days: HtSprintDay[];
    participants: HtParticipant[];
};

export type HtTrackerProps = {
    sprint: HtSprintMeta | null;
    days: HtSprintDay[];
    participants: HtParticipant[];
    today: string;
    flash?: HtFlash;
};

export type HtHistoryIndexProps = {
    sprints: Array<{
        id: number;
        start_date: string;
        end_date: string;
        ended_at: string | null;
        participant_count: number;
        achievers: Array<{
            person_id: number;
            name: string;
            reward_name: string;
        }>;
    }>;
};

export type HtHistoryShowProps = {
    sprint: HtSprintMeta;
    days: HtSprintDay[];
    participants: HtParticipant[];
    today: string;
};

export type HtTotalsProps = {
    people: Array<{
        person_id: number;
        name: string;
        display_order: number;
        total_ticks_earned: number;
        total_rewards_achieved: number;
        sprints_participated_count: number;
    }>;
};

export type HtNextSprintParticipant = {
    person_id: number;
    person_name: string;
    habits: Array<{ name: string; display_order: number }>;
    active_reward: { id: number; name: string; cost: number } | null;
};

export type HtSettingsProps = {
    people: HtPersonWithTrash[];
    active_sprint: HtGrid | null;
    available_for_sprint: HtPerson[];
    rewards_by_person: Record<number, HtReward[]>;
    next_sprint_defaults: {
        start_date: string;
        end_date: string;
        participants: HtNextSprintParticipant[];
    };
    today: string;
};
