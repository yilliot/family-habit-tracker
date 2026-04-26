<?php

namespace App\Http\Controllers\HabitTracker;

use App\Actions\HabitTracker\CreateReward;
use App\Actions\HabitTracker\DeleteReward;
use App\Actions\HabitTracker\UpdateReward;
use App\Http\Controllers\Controller;
use App\Http\Requests\HabitTracker\StoreRewardRequest;
use App\Http\Requests\HabitTracker\UpdateRewardRequest;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Http\RedirectResponse;

class RewardController extends Controller
{
    public function __construct(
        private CreateReward $createReward,
        private UpdateReward $updateReward,
        private DeleteReward $deleteReward,
    ) {}

    /**
     * Create a reward for a person.
     */
    public function store(StoreRewardRequest $request, Person $person): RedirectResponse
    {
        $this->createReward->execute(
            $person,
            $request->string('name')->toString(),
            $request->integer('cost'),
        );

        return to_route('settings.index')->with('success', 'Reward set');
    }

    /**
     * Update a reward's name and/or cost.
     *
     * If the cost change tripped an immediate achievement on the active-sprint
     * participant linked to this reward, surface a `flash.achievement` so the
     * Frontend can open the celebration overlay.
     */
    public function update(UpdateRewardRequest $request, Reward $reward): RedirectResponse
    {
        $wasAchieved = $reward->isAchieved();

        $updated = $this->updateReward->execute($reward, $request->validated());

        $redirect = to_route('settings.index')->with('success', 'Reward updated');

        if (! $wasAchieved && $updated->isAchieved()) {
            $participant = SprintParticipant::query()
                ->where('active_reward_id', $updated->id)
                ->whereIn('sprint_id', Sprint::query()->active()->select('id'))
                ->with('person')
                ->first();

            if ($participant !== null) {
                $redirect = $redirect->with('achievement', [
                    'person_id' => $participant->person->id,
                    'reward_id' => $updated->id,
                    'person_name' => $participant->person->name,
                    'reward_name' => $updated->name,
                ]);
            }
        }

        return $redirect;
    }

    /**
     * Delete an unachieved, unlinked reward.
     */
    public function destroy(Reward $reward): RedirectResponse
    {
        $this->deleteReward->execute($reward);

        return to_route('settings.index')->with('success', 'Reward deleted');
    }
}
