<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\AddParticipant;
use App\Actions\HabitTracker\UpdateParticipantReward;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StoreSprintParticipantRequest;
use App\Http\Requests\HabitTracker\UpdateParticipantRewardRequest;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Http\RedirectResponse;

class SprintParticipantController extends Controller
{
    public function __construct(
        private AddParticipant $addParticipant,
        private UpdateParticipantReward $updateParticipantReward,
    ) {}

    /**
     * Add a participant mid-sprint.
     */
    public function store(StoreSprintParticipantRequest $request, Sprint $sprint): RedirectResponse
    {
        $payload = $request->validated();

        $habits = array_map(
            fn (array $h): array => [
                'name' => $h['name'],
                'display_order' => $h['display_order'] ?? null,
            ],
            $payload['habits'] ?? [],
        );

        $reward = $this->resolveRewardInput($payload['reward'] ?? null);

        $this->addParticipant->execute(
            $sprint,
            (int) $payload['person_id'],
            $habits,
            $reward,
        );

        return to_route('settings.index')->with('success', 'Participant added');
    }

    /**
     * Swap which reward is the "active" one for this participant.
     */
    public function updateReward(
        UpdateParticipantRewardRequest $request,
        SprintParticipant $sprintParticipant,
    ): RedirectResponse {
        $this->updateParticipantReward->execute(
            $sprintParticipant,
            $request->integer('reward_id'),
        );

        return to_route('settings.index')->with('success', 'Active reward updated');
    }

    /**
     * @param  array<string, mixed>|null  $reward
     * @return array{name:string, cost:int}|null
     */
    private function resolveRewardInput(?array $reward): ?array
    {
        if ($reward === null) {
            return null;
        }

        $mode = $reward['mode'] ?? 'keep_current';

        if ($mode === 'new') {
            return [
                'name' => $reward['name'],
                'cost' => (int) $reward['cost'],
            ];
        }

        return null;
    }
}
