<?php

namespace App\Models;

use Database\Factories\PersonFactory;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

class Person extends Model
{
    /** @use HasFactory<PersonFactory> */
    use HasFactory, SoftDeletes;

    protected $table = 'people';

    protected $fillable = [
        'name',
        'display_order',
    ];

    protected $attributes = [
        'display_order' => 0,
    ];

    /**
     * Validation rules for the `name` attribute, exposed for re-use by
     * Platform-layer Form Requests.
     *
     * @return array<int, string>
     */
    public static function nameRules(): array
    {
        return ['required', 'string', 'min:1', 'max:50'];
    }

    /**
     * Validation rules for the `display_order` attribute.
     *
     * @return array<int, string>
     */
    public static function displayOrderRules(): array
    {
        return ['nullable', 'integer', 'min:0'];
    }

    /**
     * @return HasMany<SprintParticipant, $this>
     */
    public function sprintParticipants(): HasMany
    {
        return $this->hasMany(SprintParticipant::class);
    }

    /**
     * @return HasMany<Reward, $this>
     */
    public function rewards(): HasMany
    {
        return $this->hasMany(Reward::class);
    }

    /**
     * @param  Builder<Person>  $query
     */
    public function scopeOrdered(Builder $query): void
    {
        $query->orderBy('display_order')->orderBy('id');
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'display_order' => 'integer',
        ];
    }
}
