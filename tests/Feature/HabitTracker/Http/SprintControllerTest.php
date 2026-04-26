<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Person;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class SprintControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_store_starts_a_new_sprint_with_participants(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $father = Person::factory()->create(['name' => '爸爸', 'display_order' => 1]);
        $mother = Person::factory()->create(['name' => '妈妈', 'display_order' => 2]);

        $this->from(route('settings.index'))
            ->post(route('sprints.store'), [
                'start_date' => '2026-04-19',
                'end_date' => '2026-05-16',
                'participants' => [
                    [
                        'person_id' => $father->id,
                        'habits' => [['name' => 'Read 30min']],
                        'reward' => ['mode' => 'new', 'name' => 'Coffee Mug', 'cost' => 50],
                    ],
                    [
                        'person_id' => $mother->id,
                        'habits' => [['name' => 'Walk']],
                        'reward' => ['mode' => 'none'],
                    ],
                ],
            ])
            ->assertRedirect(route('tracker.show'))
            ->assertSessionHas('success', 'Sprint started');

        $this->assertDatabaseHas('sprints', [
            'status' => Sprint::STATUS_ACTIVE,
        ]);
        $this->assertSame(1, Sprint::query()->whereDate('start_date', '2026-04-19')->count());
    }

    public function test_store_rejects_when_active_sprint_exists(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();
        Sprint::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprints.store'), [
                'start_date' => '2026-04-19',
                'end_date' => '2026-05-16',
                'participants' => [
                    ['person_id' => $person->id, 'habits' => [], 'reward' => ['mode' => 'none']],
                ],
            ])
            ->assertSessionHasErrors('sprint');
    }

    public function test_store_rejects_end_before_start(): void
    {
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprints.store'), [
                'start_date' => '2026-05-16',
                'end_date' => '2026-04-19',
                'participants' => [
                    ['person_id' => $person->id, 'habits' => [], 'reward' => ['mode' => 'none']],
                ],
            ])
            ->assertSessionHasErrors('end_date');
    }

    public function test_store_rejects_empty_participants(): void
    {
        $this->from(route('settings.index'))
            ->post(route('sprints.store'), [
                'start_date' => '2026-04-19',
                'end_date' => '2026-05-16',
                'participants' => [],
            ])
            ->assertSessionHasErrors('participants');
    }

    public function test_store_requires_reward_name_and_cost_when_mode_new(): void
    {
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->post(route('sprints.store'), [
                'start_date' => '2026-04-19',
                'end_date' => '2026-05-16',
                'participants' => [
                    [
                        'person_id' => $person->id,
                        'habits' => [],
                        'reward' => ['mode' => 'new'],
                    ],
                ],
            ])
            ->assertSessionHasErrors([
                'participants.0.reward.name',
                'participants.0.reward.cost',
            ]);
    }

    public function test_end_archives_active_sprint(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $sprint = Sprint::factory()->create();
        SprintParticipant::factory()->for($sprint)->for(Person::factory()->create())->create();

        $this->from(route('tracker.show'))
            ->post(route('sprints.end', $sprint))
            ->assertRedirect(route('tracker.show'))
            ->assertSessionHas('success', 'Sprint archived. You can start a new one.');

        $this->assertSame(Sprint::STATUS_ARCHIVED, $sprint->refresh()->status);
    }

    public function test_end_rejects_already_archived(): void
    {
        $sprint = Sprint::factory()->archived()->create();

        $this->from(route('tracker.show'))
            ->post(route('sprints.end', $sprint))
            ->assertSessionHasErrors('sprint');
    }
}
