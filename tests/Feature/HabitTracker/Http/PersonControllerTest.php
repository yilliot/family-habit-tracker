<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Person;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Tests\TestCase;

class PersonControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_store_creates_person(): void
    {
        $this->from(route('settings.index'))
            ->post(route('people.store'), [
                'name' => 'Charlie',
                'display_order' => 5,
            ])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Person added');

        $this->assertDatabaseHas('people', [
            'name' => 'Charlie',
            'display_order' => 5,
        ]);
    }

    public function test_store_validation_rejects_blank_name(): void
    {
        $this->from(route('settings.index'))
            ->post(route('people.store'), ['name' => ''])
            ->assertSessionHasErrors('name');
    }

    public function test_store_validation_rejects_too_long_name(): void
    {
        $this->from(route('settings.index'))
            ->post(route('people.store'), ['name' => str_repeat('a', 51)])
            ->assertSessionHasErrors('name');
    }

    public function test_update_changes_name(): void
    {
        $person = Person::factory()->create(['name' => 'Old']);

        $this->from(route('settings.index'))
            ->patch(route('people.update', $person), ['name' => 'New'])
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Person updated');

        $this->assertSame('New', $person->refresh()->name);
    }

    public function test_destroy_soft_deletes_person(): void
    {
        $person = Person::factory()->create();

        $this->from(route('settings.index'))
            ->delete(route('people.destroy', $person))
            ->assertRedirect(route('settings.index'))
            ->assertSessionHas('success', 'Person removed');

        $this->assertSoftDeleted('people', ['id' => $person->id]);
    }
}
